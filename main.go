package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	g "github.com/AllenDang/giu"
	"github.com/go-audio/wav"
	"github.com/xyproto/synth"
)

const (
	versionString   = "Kickpad 0.0.1"
	buttonSize      = 100
	numPads         = 16
	maxGenerations  = 1000
	maxStagnation   = 50 // Stop after 50 generations with no fitness improvement
	defaultBitDepth = 16
)

var (
	activePadIndex  int
	pads            [numPads]*synth.Settings
	loadedWaveform  []float64                 // Loaded .wav file waveform data
	trainingOngoing bool                      // Indicates whether the GA is running
	wavFilePath     string    = "kick808.wav" // Default .wav file path
	statusMessage   string                    // Status message displayed at the bottom
	cancelTraining  chan bool                 // Channel to cancel GA training
	sampleRates               = []int{44100, 48000, 96000, 192000}
	bitDepth        int       = defaultBitDepth
	sampleRate      int       = sampleRates[0]
)

// Dropdown selection index for the waveform
var waveformSelectedIndex int32
var sampleRateIndex int32
var bitDepthSelected bool

// Load a .wav file and store the waveform for comparison
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

// Play the loaded .wav file using a command-line player
func playWavFile() error {
	workingDir, err := os.Getwd()
	if err != nil {
		statusMessage = "Error: Could not get working directory"
		return err
	}

	filePath := filepath.Join(workingDir, wavFilePath)

	if err := synth.FFPlayWav(filePath); err != nil {
		statusMessage = fmt.Sprintf("Error: Failed to play .wav file %s", filePath)
		return err
	}

	statusMessage = fmt.Sprintf("Playing .wav file: %s", filePath)
	return nil
}

// Compare two waveforms using Mean Squared Error (MSE)
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

// Randomize all pads (instead of mutating)
func randomizeAllPads() {
	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandom(nil, sampleRate, bitDepth)
	}
}

// Function to optimize the settings using a genetic algorithm without writing to disk
func optimizeSettings() {
	if len(loadedWaveform) == 0 {
		statusMessage = "Error: No .wav file loaded. Please load a .wav file first."
		return
	}

	// Display initial status message before starting the training
	statusMessage = "Training started..."
	g.Update()

	population := make([]*synth.Settings, 100)
	for i := 0; i < len(population); i++ {
		population[i] = synth.NewRandom(nil, sampleRate, bitDepth)
	}

	bestSettings := pads[activePadIndex]
	bestFitness := math.Inf(1)
	stagnationCount := 0

	for generation := 0; generation < maxGenerations && trainingOngoing; generation++ {
		select {
		case <-cancelTraining:
			statusMessage = "Training canceled."
			trainingOngoing = false
			return
		default:
		}

		improved := false

		for _, individual := range population {
			// Generate the kick in memory
			generatedWaveform, err := individual.GenerateKickWaveform()
			if err != nil {
				statusMessage = "Error: Failed to generate kick."
				continue
			}

			// Compare the generated waveform with the target waveform
			fitness := compareWaveforms(generatedWaveform, loadedWaveform)

			// Update best config if the fitness is better
			if fitness < bestFitness {
				bestFitness = fitness
				bestSettings = synth.CopySettings(individual)
				pads[activePadIndex] = bestSettings // Save the best result to the active pad
				improved = true

				if bestFitness < 1e-3 {
					statusMessage = fmt.Sprintf("Global optimum found at generation %d!", generation)
					trainingOngoing = false
					return
				}
			}
		}

		if !improved {
			stagnationCount++
			if stagnationCount >= maxStagnation {
				statusMessage = "Training stopped due to no improvement in 50 generations."
				trainingOngoing = false
				return
			}
		} else {
			stagnationCount = 0
		}

		for i := 0; i < len(population); i++ {
			mutateSettings(population[i])
		}

		statusMessage = fmt.Sprintf("Generation %d: Best fitness = %f", generation, bestFitness)
		g.Update() // Update UI with the new generation status
	}

	// After training, save the best result to the active pad and update the sliders
	pads[activePadIndex] = bestSettings
	g.Update() // Update the sliders with the best result
}

// Function to mutate a single config
func mutateSettings(cfg *synth.Settings) {
	mutationFactor := 0.01

	// Mutate the configuration
	cfg.Attack *= (0.8 + rand.Float64()*0.4)
	cfg.Decay *= (0.8 + rand.Float64()*0.4)
	cfg.Sustain *= (0.8 + rand.Float64()*0.4)
	cfg.Release *= (0.8 + rand.Float64()*0.4)
	cfg.Drive *= (0.8 + rand.Float64()*0.4)
	cfg.FilterCutoff *= (0.8 + rand.Float64()*0.4)
	cfg.Sweep *= (0.8 + rand.Float64()*0.4)
	cfg.PitchDecay *= (0.8 + rand.Float64()*0.4)

	// Mutate the waveform and noise amounts
	if rand.Float64() < mutationFactor {
		cfg.WaveformType = rand.Intn(7) // Mutate to any waveform type
	}
	if rand.Float64() < mutationFactor {
		cfg.NoiseAmount *= (0.8 + rand.Float64()*0.4)
	}
}

// UI functions and pad widget handling
func createPadWidget(cfg *synth.Settings, padLabel string, padIndex int) g.Widget {
	return g.Style().SetColor(g.StyleColorButton, cfg.Color()).To(
		g.Column(
			g.Button(padLabel).Size(buttonSize, buttonSize).OnClick(func() {
				// Clear the status message when a pad is clicked
				statusMessage = ""
				// Set the clicked pad as active
				activePadIndex = padIndex
				// Then generate and play the sample (even during training)
				go func() {
					err := synth.FFPlayKick(pads[activePadIndex])
					if err != nil {
						statusMessage = fmt.Sprintf("Error: Failed to play kick: %v", err)
					}
				}()
			}),
			// Mutate button: Mutate the selected pad and update the sliders
			g.Button("Mutate").OnClick(func() {
				mutateSettings(pads[padIndex]) // Mutate the selected pad and update settings
				activePadIndex = padIndex
				g.Update() // Update the sliders with mutated settings
			}),
			// Save button: Save the current pad's configuration as a .wav file
			g.Button("Save").OnClick(func() {
				fileName, err := pads[padIndex].SaveKickTo(".") // Save the active pad's settings to a .wav file
				if err != nil {
					statusMessage = fmt.Sprintf("Error: Failed to save kick to %s", ".")
				} else {
					statusMessage = fmt.Sprintf("Kick saved to %s", fileName)
				}
				g.Update() // Update the status message
			}),
		),
	)
}

// Function to create sliders and dropdown for viewing and editing the selected pad's settings
func createSlidersForSelectedPad() g.Widget {
	cfg := pads[activePadIndex]

	// Convert float64 to float32 for sliders
	attack := float32(cfg.Attack)
	decay := float32(cfg.Decay)
	sustain := float32(cfg.Sustain)
	release := float32(cfg.Release)
	drive := float32(cfg.Drive)
	filterCutoff := float32(cfg.FilterCutoff)
	sweep := float32(cfg.Sweep)
	pitchDecay := float32(cfg.PitchDecay)

	// List of available waveforms
	waveforms := []string{"Sine", "Triangle", "Sawtooth", "Square", "Noise White", "Noise Pink", "Noise Brown"}
	waveformSelectedIndex = int32(cfg.WaveformType)

	return g.Column(
		g.Label(fmt.Sprintf("Kick Pad %d settings:", activePadIndex+1)),
		g.Dummy(30, 0),
		g.Row(
			g.Label("Waveform"),
			g.Combo("Waveform", waveforms[waveformSelectedIndex], waveforms, &waveformSelectedIndex).Size(150).OnChange(func() {
				cfg.WaveformType = int(waveformSelectedIndex)
			}),
		),
		g.Row(
			g.Label("Attack"),
			g.SliderFloat(&attack, 0.0, 1.0).Size(150).OnChange(func() { cfg.Attack = float64(attack) }),
		),
		g.Row(
			g.Label("Decay"),
			g.SliderFloat(&decay, 0.1, 1.0).Size(150).OnChange(func() { cfg.Decay = float64(decay) }),
		),
		g.Row(
			g.Label("Sustain"),
			g.SliderFloat(&sustain, 0.0, 1.0).Size(150).OnChange(func() { cfg.Sustain = float64(sustain) }),
		),
		g.Row(
			g.Label("Release"),
			g.SliderFloat(&release, 0.1, 1.0).Size(150).OnChange(func() { cfg.Release = float64(release) }),
		),
		g.Row(
			g.Label("Drive"),
			g.SliderFloat(&drive, 0.0, 1.0).Size(150).OnChange(func() { cfg.Drive = float64(drive) }),
		),
		g.Row(
			g.Label("Filter Cutoff"),
			g.SliderFloat(&filterCutoff, 1000, 8000).Size(150).OnChange(func() { cfg.FilterCutoff = float64(filterCutoff) }),
		),
		g.Row(
			g.Label("Sweep"),
			g.SliderFloat(&sweep, 0.1, 2.0).Size(150).OnChange(func() { cfg.Sweep = float64(sweep) }),
		),
		g.Row(
			g.Label("Pitch Decay"),
			g.SliderFloat(&pitchDecay, 0.1, 1.5).Size(150).OnChange(func() { cfg.PitchDecay = float64(pitchDecay) }),
		),
		g.Dummy(30, 0),
		g.Row(
			g.Label("Sample Rate"),
			g.Combo("Sample Rate", fmt.Sprintf("%d Hz", sampleRates[sampleRateIndex]), []string{
				"44100 Hz", "48000 Hz", "96000 Hz", "192000 Hz",
			}, &sampleRateIndex).Size(150).OnChange(func() {
				sampleRate = sampleRates[sampleRateIndex]
			}),
		),
		g.Row(
			g.Label("Bit Depth"),
			g.Checkbox("24-bit", &bitDepthSelected).OnChange(func() {
				if bitDepthSelected {
					bitDepth = 24
				} else {
					bitDepth = 16
				}
			}),
		),
		g.Dummy(30, 0),
		g.Row(
			g.Button("Play").OnClick(func() {
				statusMessage = ""
				err := synth.FFPlayKick(pads[activePadIndex])
				if err != nil {
					statusMessage = "Error: Failed to play kick."
				}
				g.Update() // Update the status message
			}),
			g.Button("Randomize all").OnClick(func() {
				randomizeAllPads() // Randomize all pads
				g.Update()         // Refresh the UI with randomized settings
			}),
			g.Button("Save").OnClick(func() {
				fileName, err := pads[activePadIndex].SaveKickTo(".")
				if err != nil {
					statusMessage = fmt.Sprintf("Error: Failed to save kick to %s", fileName)
				} else {
					statusMessage = fmt.Sprintf("Kick saved to %s", fileName)
				}
				g.Update() // Update the status message
			}),
		),
	)
}

// Function to create the UI layout
func loop() {
	// Display the 16 pads in a 4x4 grid
	padGrid := []g.Widget{}
	padIndex := 0
	for row := 0; row < 4; row++ {
		rowWidgets := []g.Widget{}
		for col := 0; col < 4; col++ {
			rowWidgets = append(rowWidgets, createPadWidget(pads[padIndex], fmt.Sprintf("Pad %d", padIndex+1), padIndex))
			padIndex++
		}
		padGrid = append(padGrid, g.Row(rowWidgets...))
	}

	// Build the layout with the grid on the left, sliders and buttons on the right, and text input for the .wav file path below
	g.SingleWindow().Layout(
		g.Row(
			g.Column(padGrid...),
			g.Column(
				createSlidersForSelectedPad(),
				g.Dummy(30, 0),
				g.Row(
					g.InputText(&wavFilePath).Size(200), // Default text is "kick808.wav"
					g.Button("Load WAV").OnClick(func() {
						err := loadWavFile()
						if err != nil {
							statusMessage = "Failed to load .wav file"
						}
					}),
				),
				// Use g.Condition before g.Row to ensure that rows aren't empty
				g.Condition(len(loadedWaveform) > 0 || trainingOngoing,
					g.Layout{
						g.Row(generateTrainingButtons()),
					},
					nil,
				),
			),
		),
		// Status message label at the bottom
		g.Label(statusMessage),
	)
}

// Conditionally generate the "Find kick similar to WAV" and "Stop training" buttons
func generateTrainingButtons() g.Widget {
	if len(loadedWaveform) > 0 {
		if trainingOngoing {
			return g.Row(
				g.Button("Stop training").OnClick(func() {
					if trainingOngoing {
						cancelTraining <- true
					}
				}),
				g.Button("Play WAV").OnClick(func() {
					err := playWavFile()
					if err != nil {
						statusMessage = "Error: Failed to play WAV"
					}
					g.Update() // Update status message for WAV playback
				}),
			)
		}
		return g.Row(
			g.Button("Find kick similar to WAV").OnClick(func() {
				if !trainingOngoing {
					cancelTraining = make(chan bool)
					trainingOngoing = true
					go optimizeSettings() // Run the optimization in a goroutine
				}
			}),
			g.Button("Play WAV").OnClick(func() {
				err := playWavFile()
				if err != nil {
					statusMessage = "Error: Failed to play WAV"
				}
				g.Update() // Update status message for WAV playback
			}),
		)
	}
	return g.Dummy(0, 0) // Dummy widget if no buttons should be shown
}

func main() {
	// Initialize random settings for the 16 pads using synth.NewRandom()
	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandom(nil, sampleRate, bitDepth)
	}

	// Set the first pad as selected
	activePadIndex = 0

	// Adjust the window size to fit the grid, buttons, and sliders better
	wnd := g.NewMasterWindow(versionString, 780, 660, g.MasterWindowFlagsNotResizable)
	wnd.Run(loop)
}
