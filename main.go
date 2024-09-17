package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/go-audio/wav"
	"github.com/xyproto/synth"
)

const (
	numPads        = 16
	maxGenerations = 1000
	maxStagnation  = 50 // Stop after 50 generations with no fitness improvement
)

var (
	activePadIndex  int
	pads            [numPads]*synth.Settings
	loadedWaveform  []float64                 // Loaded .wav file waveform data
	trainingOngoing bool                      // Indicates whether the GA is running
	wavFilePath     string    = "kick808.wav" // Default .wav file path
	statusMessage   string                    // Status message displayed at the bottom
	cancelTraining  chan bool                 // Channel to cancel GA training
	sdl2            *synth.SDL2
)

func init() {
	sdl2 = synth.NewSDL2()
}

func loadWavFile() error {
	workingDir, err := os.Getwd()
	if err != nil {
		statusMessage = "Error: Could not get working directory"
		return err
	}

	filePath := filepath.Join(workingDir, wavFilePath)
	file, err := os.Open(filePath)
	if err != nil {
		statusMessage = fmt.Sprintf("Error: Failed to open .wav file %s", filePath)
		return err
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		statusMessage = fmt.Sprintf("Error: Failed to decode .wav file %s", filePath)
		return err
	}

	loadedWaveform = make([]float64, len(buffer.Data))
	for i, sample := range buffer.Data {
		loadedWaveform[i] = float64(sample) / math.MaxInt16 // Normalize to [-1, 1] range
	}

	statusMessage = fmt.Sprintf("Loaded .wav file: %s", filePath)
	return nil
}

func playWavFile() error {
	workingDir, err := os.Getwd()
	if err != nil {
		statusMessage = "Error: Could not get working directory"
		return err
	}

	filePath := filepath.Join(workingDir, wavFilePath)
	err = synth.FFPlayWav(filePath)
	if err != nil {
		statusMessage = fmt.Sprintf("Error: Failed to play .wav file %s", filePath)
		return err
	}

	statusMessage = fmt.Sprintf("Playing .wav file: %s", filePath)
	return nil
}

func compareWaveforms(waveform1, waveform2 []float64) float64 {
	minLength := len(waveform1)
	if len(waveform2) < minLength {
		minLength = len(waveform2)
	}

	mse := 0.0
	for i := 0; i < minLength; i++ {
		diff := waveform1[i] - waveform2[i]
		mse += diff * diff
	}
	return mse / float64(minLength)
}

func randomizeAllPads() {
	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandom(nil)
	}
}

func mutateSettings(cfg *synth.Settings) {
	mutationFactor := 0.01
	cfg.Attack *= (0.8 + rand.Float64()*0.4)
	cfg.Decay *= (0.8 + rand.Float64()*0.4)
	cfg.Sustain *= (0.8 + rand.Float64()*0.4)
	cfg.Release *= (0.8 + rand.Float64()*0.4)
	cfg.Drive *= (0.8 + rand.Float64()*0.4)
	cfg.FilterCutoff *= (0.8 + rand.Float64()*0.4)
	cfg.Sweep *= (0.8 + rand.Float64()*0.4)
	cfg.PitchDecay *= (0.8 + rand.Float64()*0.4)

	if rand.Float64() < mutationFactor {
		cfg.WaveformType = rand.Intn(7)
	}
	if rand.Float64() < mutationFactor {
		cfg.NoiseAmount *= (0.8 + rand.Float64()*0.4)
	}
}

func optimizeSettings(statusLabel *widget.Label) {
	if len(loadedWaveform) == 0 {
		statusLabel.SetText("Error: No .wav file loaded. Please load a .wav file first.")
		return
	}

	statusLabel.SetText("Training started...")

	population := make([]*synth.Settings, 100)
	for i := 0; i < len(population); i++ {
		population[i] = synth.NewRandom(nil)
	}

	bestSettings := pads[activePadIndex]
	bestFitness := math.Inf(1)
	stagnationCount := 0

	for generation := 0; generation < maxGenerations && trainingOngoing; generation++ {
		select {
		case <-cancelTraining:
			statusLabel.SetText("Training canceled.")
			trainingOngoing = false
			return
		default:
		}

		improved := false

		for _, individual := range population {
			generatedWaveform, err := individual.GenerateKickWaveform()
			if err != nil {
				statusLabel.SetText("Error: Failed to generate kick.")
				continue
			}

			fitness := compareWaveforms(generatedWaveform, loadedWaveform)
			if fitness < bestFitness {
				bestFitness = fitness
				bestSettings = synth.CopySettings(individual)
				pads[activePadIndex] = bestSettings
				improved = true

				if bestFitness < 1e-3 {
					statusLabel.SetText(fmt.Sprintf("Global optimum found at generation %d!", generation))
					trainingOngoing = false
					return
				}
			}
		}

		if !improved {
			stagnationCount++
			if stagnationCount >= maxStagnation {
				statusLabel.SetText("Training stopped due to no improvement in 50 generations.")
				trainingOngoing = false
				return
			}
		} else {
			stagnationCount = 0
		}

		for i := 0; i < len(population); i++ {
			mutateSettings(population[i])
		}

		statusLabel.SetText(fmt.Sprintf("Generation %d: Best fitness = %f", generation, bestFitness))
	}
}

func createPadButton(padLabel string, padIndex int, window fyne.Window, statusLabel *widget.Label) *widget.Button {
	return widget.NewButton(padLabel, func() {
		activePadIndex = padIndex
		go func() {
			err := sdl2.PlayKick(pads[padIndex])
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error: Failed to play kick: %v", err))
			}
		}()
	})
}

func createMainWindow(app fyne.App) {
	window := app.NewWindow("Kick Drum Generator")

	statusLabel := widget.NewLabel("")

	// Create a grid of buttons for the 16 pads
	var buttons []fyne.CanvasObject // Change this from []*fyne.Container to []fyne.CanvasObject
	for i := 0; i < numPads; i++ {
		buttons = append(buttons, container.NewVBox(
			createPadButton(fmt.Sprintf("Pad %d", i+1), i, window, statusLabel),
			widget.NewButton("Mutate", func() {
				mutateSettings(pads[i])
			}),
			widget.NewButton("Save", func() {
				fileName, err := pads[i].SaveKickTo(".")
				if err != nil {
					statusLabel.SetText(fmt.Sprintf("Error: Failed to save kick to %s", "."))
				} else {
					statusLabel.SetText(fmt.Sprintf("Kick saved to %s", fileName))
				}
			}),
		))
	}

	// Layout for sliders and file input
	wavInput := widget.NewEntry()
	wavInput.SetPlaceHolder("kick808.wav")
	wavInput.OnChanged = func(path string) {
		wavFilePath = path
	}

	// WAV load and play buttons
	loadButton := widget.NewButton("Load WAV", func() {
		err := loadWavFile()
		if err != nil {
			statusLabel.SetText("Failed to load .wav file")
		}
	})
	playButton := widget.NewButton("Play WAV", func() {
		err := playWavFile()
		if err != nil {
			statusLabel.SetText("Error: Failed to play WAV")
		}
	})

	// Randomize button
	randomizeButton := widget.NewButton("Randomize all", func() {
		randomizeAllPads()
	})

	// Use fyne.CanvasObject slice for the grid layout
	window.SetContent(container.NewVBox(
		container.NewGridWithColumns(4, buttons...), // Fixed to use []fyne.CanvasObject
		wavInput,
		loadButton,
		playButton,
		randomizeButton,
		statusLabel,
	))

	window.Resize(fyne.NewSize(800, 600))
	window.ShowAndRun()
}

func main() {
	// Initialize the synth pads
	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandom(nil)
	}

	// Create a new Fyne application and run the GUI
	app := app.New()
	createMainWindow(app)
}
