package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	g "github.com/AllenDang/giu"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
	"github.com/xyproto/playsample"
	"github.com/xyproto/synth"
)

const (
	versionString     = "Kickpad 1.5.0"
	channels          = 2
	numPads           = 16
	maxGenerations    = 1000
	maxStagnation     = 10
	defaultBitDepth   = 16
	populationSize    = 100
	tournamentSize    = 5
	eliteCount        = 10
	mutationRate      = 0.05
	minAttack         = 0.05
	maxAttack         = 0.5
	minDecay          = 0.05
	maxDecay          = 0.5
	minSustain        = 0.1
	maxSustain        = 1.0
	minRelease        = 0.05
	maxRelease        = 1.0
	minDrive          = 0.0
	maxDrive          = 1.0
	minFilterCutoff   = 500
	maxFilterCutoff   = 10000
	minSweep          = 0.1
	maxSweep          = 2.0
	minPitchDecay     = 0.1
	maxPitchDecay     = 1.5
	minNoiseAmount    = 0.0
	maxNoiseAmount    = 1.0
	minSampleDuration = 0.1
	maxSampleDuration = 2.0
)

var (
	soundTypes                   = []string{"kick", "clap", "snare", "closed_hh", "open_hh", "rimshot", "tom", "percussion", "ride", "crash", "bass", "xylophone", "lead"}
	soundTypeSelectedIndex int32 = 0
	activePadIndex         int
	pads                   [numPads]*synth.Settings
	padSoundTypes          = make([]string, numPads)
	loadedWaveform         []float64
	trainingOngoing        bool
	wavFilePath            string = "kick909.wav"
	statusMessage          string
	cancelTraining         chan bool
	sampleRates                = []int{44100, 48000, 96000, 192000}
	bitDepth               int = defaultBitDepth
	sampleRate             int = sampleRates[0]
	waveformSelectedIndex  int32
	sampleRateIndex        int32
	bitDepthSelected       bool
	player                 *playsample.Player
	muPlayer               sync.Mutex
	mu                     sync.Mutex
)

func setStatusMessage(msg string) {
	mu.Lock()
	defer mu.Unlock()
	statusMessage = msg
}

func GeneratePlay(t string, cfg *synth.Settings) error {
	muPlayer.Lock()
	defer muPlayer.Unlock()

	if player == nil || !player.Initialized {
		return errors.New("Player is not initialized")
	}

	samples, err := cfg.Generate(t)
	if err != nil {
		return err
	}
	return player.PlayWaveform(samples, cfg.SampleRate, cfg.BitDepth, cfg.Channels)
}

func loadWavFile() error {
	workingDir, err := os.Getwd()
	if err != nil {
		setStatusMessage("Error: Could not get working directory")
		return err
	}

	filePath := filepath.Join(workingDir, wavFilePath)
	file, err := os.Open(filePath)
	if err != nil {
		setStatusMessage(fmt.Sprintf("Error: Failed to open .wav file %s", filePath))
		return err
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		setStatusMessage(fmt.Sprintf("Error: Failed to decode .wav file %s", filePath))
		return err
	}

	loadedWaveform = make([]float64, len(buffer.Data))
	for i, sample := range buffer.Data {
		loadedWaveform[i] = float64(sample) / 32768
	}

	setStatusMessage(fmt.Sprintf("Loaded .wav file: %s", filePath))
	return nil
}

func playWavFile() error {
	workingDir, err := os.Getwd()
	if err != nil {
		setStatusMessage("Error: Could not get working directory")
		return err
	}

	filePath := filepath.Join(workingDir, wavFilePath)

	muPlayer.Lock()
	defer muPlayer.Unlock()
	if player == nil || !player.Initialized {
		return errors.New("Player is not initialized")
	}

	if err := player.PlayWav(filePath); err != nil {
		setStatusMessage(fmt.Sprintf("Error: Failed to play .wav file %s", filePath))
		return err
	}

	setStatusMessage(fmt.Sprintf("Playing .wav file: %s", filePath))
	return nil
}

func createPadWidget(cfg *synth.Settings, padLabel string, padIndex int) g.Widget {
	buttonColor := cfg.Color()
	padBorderColor := color.RGBA{0x0, 0x0, 0x0, 0xff}
	padTextColor := color.RGBA{0x0, 0x0, 0x0, 0xff}

	luminance := 0.299*float64(buttonColor.R) + 0.587*float64(buttonColor.G) + 0.114*float64(buttonColor.B)
	if luminance < 128 {
		padTextColor = color.RGBA{0xff, 0xff, 0xff, 0xff}
	}

	if padIndex == activePadIndex {
		padBorderColor = color.RGBA{0xff, 0x0, 0x0, 0xff}
	}

	return g.Style().SetColor(g.StyleColorButton, buttonColor).To(
		g.Column(
			g.Style().SetColor(g.StyleColorText, padTextColor).SetColor(g.StyleColorBorder, padBorderColor).To(
				g.Button(padLabel).Size(100, 100).OnClick(func() {
					activePadIndex = padIndex
					setStatusMessage("")
					go func() {
						if err := GeneratePlay(padSoundTypes[activePadIndex], pads[activePadIndex]); err != nil {
							setStatusMessage(fmt.Sprintf("Error: Failed to play sound: %v", err))
						} else {
							setStatusMessage(fmt.Sprintf("Playing sound from %s", padLabel))
						}
						g.Update()
					}()
				}),
			),
			g.Button("Mutate").OnClick(func() {
				mutateSettings(pads[padIndex], true)
				activePadIndex = padIndex
				g.Update()
			}),
			g.Button("Save").OnClick(func() {
				pads[padIndex].SampleRate = sampleRate
				pads[padIndex].BitDepth = bitDepth
				fileName, err := pads[padIndex].GenerateAndSaveTo(padSoundTypes[padIndex], ".")
				if err != nil {
					setStatusMessage(fmt.Sprintf("Error: Failed to save %s to %s", padSoundTypes[padIndex], fileName))
				} else {
					setStatusMessage(fmt.Sprintf("%s saved to %s", padSoundTypes[padIndex], fileName))
				}
				g.Update()
			}),
		),
	)
}

func createSlidersForSelectedPad() g.Widget {
	cfg := pads[activePadIndex]
	soundTypeSelectedIndex = int32(indexOfSoundType(padSoundTypes[activePadIndex]))

	attack := float32(cfg.Attack)
	decay := float32(cfg.Decay)
	sustain := float32(cfg.Sustain)
	release := float32(cfg.Release)
	drive := float32(cfg.Drive)
	filterCutoff := float32(cfg.FilterCutoff)
	sweep := float32(cfg.Sweep)
	pitchDecay := float32(cfg.PitchDecay)

	waveforms := []string{"Sine", "Triangle", "Sawtooth", "Square", "Noise White", "Noise Pink", "Noise Brown"}
	waveformSelectedIndex = int32(cfg.WaveformType)

	return g.Column(
		g.Label(fmt.Sprintf("Pad %d settings:", activePadIndex+1)),
		g.Dummy(30, 0),
		g.Row(
			g.Label("Sound Type"),
			g.Combo("Sound Type", soundTypes[soundTypeSelectedIndex], soundTypes, &soundTypeSelectedIndex).Size(150).OnChange(func() {
				padSoundTypes[activePadIndex] = soundTypes[soundTypeSelectedIndex]
				switch soundTypes[soundTypeSelectedIndex] {
				case "kick":
					pads[activePadIndex] = synth.NewRandomKick(nil, sampleRate, bitDepth, channels)
				case "snare":
					pads[activePadIndex] = synth.NewRandomSnare(nil, sampleRate, bitDepth, channels)
				default:
					pads[activePadIndex] = synth.NewRandomKick(nil, sampleRate, bitDepth, channels)
				}
				g.Update()
			}),
		),
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
			g.Checkbox("24-bit instead of 16-bit", &bitDepthSelected).OnChange(func() {
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
				setStatusMessage("")
				err := GeneratePlay(padSoundTypes[activePadIndex], pads[activePadIndex])
				if err != nil {
					setStatusMessage(fmt.Sprintf("Error: Failed to play %s.", padSoundTypes[activePadIndex]))
				}
				g.Update()
			}),
			g.Button("Randomize all").OnClick(func() {
				randomizeAllPads(sampleRate, bitDepth, channels)
				g.Update()
			}),
			g.Button("Save").OnClick(func() {
				pads[activePadIndex].SampleRate = sampleRate
				pads[activePadIndex].BitDepth = bitDepth
				fileName, err := pads[activePadIndex].GenerateAndSaveTo(padSoundTypes[activePadIndex], ".")
				if err != nil {
					setStatusMessage(fmt.Sprintf("Error: Failed to save %s to %s", padSoundTypes[activePadIndex], fileName))
				} else {
					setStatusMessage(fmt.Sprintf("%s saved to %s", padSoundTypes[activePadIndex], fileName))
				}
				g.Update()
			}),
		),
	)
}

func loop() {
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

	g.SingleWindow().Layout(
		g.Row(
			g.Column(padGrid...),
			g.Column(
				createSlidersForSelectedPad(),
				g.Dummy(30, 0),
				g.Row(
					g.InputText(&wavFilePath).Size(200),
					g.Button("Load WAV").OnClick(func() {
						err := loadWavFile()
						if err != nil {
							setStatusMessage("Failed to load .wav file")
						}
					}),
				),
				g.Condition(len(loadedWaveform) > 0 || trainingOngoing,
					g.Layout{
						g.Row(generateTrainingButtons()),
					},
					nil,
				),
			),
		),
		g.Label(statusMessage),
	)
}

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
						setStatusMessage("Error: Failed to play WAV")
					}
					g.Update()
				}),
			)
		}
		return g.Row(
			g.Button("Find sound similar to WAV").OnClick(func() {
				if !trainingOngoing {
					cancelTraining = make(chan bool)
					trainingOngoing = true
					go optimizeSettings(true)
				}
			}),
			g.Button("Play WAV").OnClick(func() {
				err := playWavFile()
				if err != nil {
					setStatusMessage("Error: Failed to play WAV")
				}
				g.Update()
			}),
		)
	}
	return g.Dummy(0, 0)
}

func optimizeSettings(allWaveforms bool) {
	if len(loadedWaveform) == 0 {
		setStatusMessage("Error: No .wav file loaded. Please load a .wav file first.")
		return
	}

	setStatusMessage("Training started...")
	g.Update()

	population := make([]*synth.Settings, populationSize)
	for i := 0; i < populationSize; i++ {
		population[i] = synth.NewRandomKick(nil, sampleRate, bitDepth, channels)

		if !allWaveforms {
			population[i].WaveformType = rand.Intn(2)
		} else {
			population[i].WaveformType = rand.Intn(7)
		}

		population[i].Attack = clamp(population[i].Attack, minAttack, maxAttack)
		population[i].Decay = clamp(population[i].Decay, minDecay, maxDecay)
		population[i].Sustain = clamp(population[i].Sustain, minSustain, maxSustain)
		population[i].Release = clamp(population[i].Release, minRelease, maxRelease)
		population[i].Drive = clamp(population[i].Drive, minDrive, maxDrive)
		population[i].FilterCutoff = clamp(population[i].FilterCutoff, minFilterCutoff, maxFilterCutoff)
		population[i].Sweep = clamp(population[i].Sweep, minSweep, maxSweep)
		population[i].PitchDecay = clamp(population[i].PitchDecay, minPitchDecay, maxPitchDecay)
		population[i].NoiseAmount = clamp(population[i].NoiseAmount, minNoiseAmount, maxNoiseAmount)
	}

	bestSettings := synth.CopySettings(population[0])
	bestFitness := compareWaveformsSafe(bestSettings)
	stagnationCount := 0

	for generation := 0; generation < maxGenerations && trainingOngoing; generation++ {
		select {
		case <-cancelTraining:
			setStatusMessage("Training canceled.")
			trainingOngoing = false
			return
		default:
		}

		fitnesses := make([]float64, populationSize)
		for i, individual := range population {
			fitnesses[i] = compareWaveformsSafe(individual)
		}

		currentBestFitness := math.Inf(1)
		currentBestIndex := -1
		for i, fitness := range fitnesses {
			if fitness < currentBestFitness {
				currentBestFitness = fitness
				currentBestIndex = i
			}
		}

		if currentBestIndex != -1 && fitnesses[currentBestIndex] < bestFitness {
			bestFitness = fitnesses[currentBestIndex]
			bestSettings = synth.CopySettings(population[currentBestIndex])
			pads[activePadIndex] = bestSettings
			pads[activePadIndex].SampleRate = sampleRate
			pads[activePadIndex].BitDepth = bitDepth
			stagnationCount = 0

			if bestFitness < 1e-3 {
				setStatusMessage(fmt.Sprintf("Global optimum found at generation %d!", generation))
				trainingOngoing = false
				return
			}
		} else {
			stagnationCount++
			if stagnationCount >= maxStagnation {
				setStatusMessage(fmt.Sprintf("Training stopped due to no improvement in %d generations.", maxStagnation))
				trainingOngoing = false
				return
			}
		}

		newPopulation := make([]*synth.Settings, 0, populationSize)

		for i := 0; i < eliteCount && i < populationSize; i++ {
			newPopulation = append(newPopulation, synth.CopySettings(bestSettings))
		}

		for len(newPopulation) < populationSize {
			parent1 := tournamentSelection(population, fitnesses, tournamentSize)
			parent2 := tournamentSelection(population, fitnesses, tournamentSize)

			child1, child2 := singlePointCrossover(parent1, parent2)

			mutateSettings(child1, allWaveforms)
			mutateSettings(child2, allWaveforms)

			newPopulation = append(newPopulation, child1, child2)
		}

		if len(newPopulation) > populationSize {
			newPopulation = newPopulation[:populationSize]
		}

		population = newPopulation

		setStatusMessage(fmt.Sprintf("Generation %d: Best fitness = %f", generation, bestFitness))
		g.Update()
	}

	pads[activePadIndex] = bestSettings
	g.Update()
}

func compareWaveformsSafe(individual *synth.Settings) float64 {
	generatedWaveform, err := individual.Generate(soundTypes[soundTypeSelectedIndex])
	if err != nil {
		return math.Inf(1)
	}

	if individual.SampleRate != sampleRate {
		generatedWaveform = synth.Resample(generatedWaveform, individual.SampleRate, sampleRate)
	}

	if originalSampleRate := 44100; sampleRate != originalSampleRate {
		loadedWaveform = synth.Resample(loadedWaveform, originalSampleRate, sampleRate)
	}

	timeMSE := compareWaveforms(generatedWaveform, loadedWaveform)
	freqMSE := compareWaveformsFFT(generatedWaveform, loadedWaveform, sampleRate)

	combinedMSE := 0.5*timeMSE + 0.5*freqMSE

	expectedDuration := individual.Attack + individual.Decay + individual.Release
	if expectedDuration < minSampleDuration {
		penalty := (minSampleDuration - expectedDuration) * 1000
		combinedMSE += penalty
	}
	if expectedDuration > maxSampleDuration {
		penalty := (expectedDuration - maxSampleDuration) * 1000
		combinedMSE += penalty
	}

	return combinedMSE
}

func compareWaveforms(waveform1, waveform2 []float64) float64 {
	if waveform1 == nil || waveform2 == nil {
		return math.Inf(1)
	}

	minLength := min(len(waveform1), len(waveform2))
	mse := 0.0
	for i := 0; i < minLength; i++ {
		diff := waveform1[i] - waveform2[i]
		mse += diff * diff
	}
	return mse / float64(minLength)
}

func compareWaveformsFFT(waveform1, waveform2 []float64, sampleRate int) float64 {
	if waveform1 == nil || waveform2 == nil {
		return math.Inf(1)
	}

	n := nextPowerOfTwo(min(len(waveform1), len(waveform2)))
	padded1 := make([]float64, n)
	padded2 := make([]float64, n)
	copy(padded1, waveform1)
	copy(padded2, waveform2)
	complex1 := fft.FFTReal(padded1)
	complex2 := fft.FFTReal(padded2)

	mag1, mag2 := make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		mag1[i] = cmplx.Abs(complex1[i])
		mag2[i] = cmplx.Abs(complex2[i])
	}

	mse := 0.0
	for i := 0; i < n; i++ {
		diff := mag1[i] - mag2[i]
		mse += diff * diff
	}
	return mse / float64(n)
}

func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	power := 1
	for power < n {
		power <<= 1
	}
	return power
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func randomizePad(padType string, sampleRate, bitDepth, channels int) *synth.Settings {
	switch padType {
	case "kick":
		return synth.NewRandomKick(nil, sampleRate, bitDepth, channels)
	case "snare":
		return synth.NewRandomSnare(nil, sampleRate, bitDepth, channels)
	default:
		return synth.NewRandomKick(nil, sampleRate, bitDepth, channels)
	}
}

func randomizeAllPads(sampleRate, bitDepth, channels int) {
	for i := 0; i < numPads; i++ {
		pads[i] = randomizePad(padSoundTypes[i], sampleRate, bitDepth, channels)
	}
}

func mutateSettings(cfg *synth.Settings, allWaveforms bool) {
	if rand.Float64() < mutationRate {
		cfg.Attack = clamp(cfg.Attack*(0.8+rand.Float64()*0.4), minAttack, maxAttack)
	}
	if rand.Float64() < mutationRate {
		cfg.Decay = clamp(cfg.Decay*(0.8+rand.Float64()*0.4), minDecay, maxDecay)
	}
	if rand.Float64() < mutationRate {
		cfg.Sustain = clamp(cfg.Sustain*(0.8+rand.Float64()*0.4), minSustain, maxSustain)
	}
	if rand.Float64() < mutationRate {
		cfg.Release = clamp(cfg.Release*(0.8+rand.Float64()*0.4), minRelease, maxRelease)
	}
	if rand.Float64() < mutationRate {
		cfg.Drive = clamp(cfg.Drive*(0.8+rand.Float64()*0.4), minDrive, maxDrive)
	}
	if rand.Float64() < mutationRate {
		cfg.FilterCutoff = clamp(cfg.FilterCutoff*(0.8+rand.Float64()*0.4), minFilterCutoff, maxFilterCutoff)
	}
	if rand.Float64() < mutationRate {
		cfg.Sweep = clamp(cfg.Sweep*(0.8+rand.Float64()*0.4), minSweep, maxSweep)
	}
	if rand.Float64() < mutationRate {
		cfg.PitchDecay = clamp(cfg.PitchDecay*(0.8+rand.Float64()*0.4), minPitchDecay, maxPitchDecay)
	}

	if rand.Float64() < mutationRate {
		if !allWaveforms {
			cfg.WaveformType = rand.Intn(2)
		} else {
			cfg.WaveformType = rand.Intn(7)
		}
	}

	if rand.Float64() < mutationRate {
		cfg.NoiseAmount = clamp(cfg.NoiseAmount*(0.8+rand.Float64()*0.4), minNoiseAmount, maxNoiseAmount)
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func singlePointCrossover(parent1, parent2 *synth.Settings) (*synth.Settings, *synth.Settings) {
	child1 := synth.CopySettings(parent1)
	child2 := synth.CopySettings(parent2)

	if rand.Float64() < 0.5 {
		child1.Attack, child2.Attack = parent2.Attack, parent1.Attack
	}
	if rand.Float64() < 0.5 {
		child1.Decay, child2.Decay = parent2.Decay, parent1.Decay
	}
	if rand.Float64() < 0.5 {
		child1.Sustain, child2.Sustain = parent2.Sustain, parent1.Sustain
	}
	if rand.Float64() < 0.5 {
		child1.Release, child2.Release = parent2.Release, parent1.Release
	}
	if rand.Float64() < 0.5 {
		child1.Drive, child2.Drive = parent2.Drive, parent1.Drive
	}
	if rand.Float64() < 0.5 {
		child1.FilterCutoff, child2.FilterCutoff = parent2.FilterCutoff, parent1.FilterCutoff
	}
	if rand.Float64() < 0.5 {
		child1.Sweep, child2.Sweep = parent2.Sweep, parent1.Sweep
	}
	if rand.Float64() < 0.5 {
		child1.PitchDecay, child2.PitchDecay = parent2.PitchDecay, parent1.PitchDecay
	}
	if rand.Float64() < 0.5 {
		child1.WaveformType, child2.WaveformType = parent2.WaveformType, parent1.WaveformType
	}
	if rand.Float64() < 0.5 {
		child1.NoiseAmount, child2.NoiseAmount = parent2.NoiseAmount, parent1.NoiseAmount
	}

	return child1, child2
}

func tournamentSelection(population []*synth.Settings, fitnesses []float64, tournamentSize int) *synth.Settings {
	bestIndex := rand.Intn(len(population))
	best := population[bestIndex]
	bestFitness := fitnesses[bestIndex]
	for i := 1; i < tournamentSize; i++ {
		competitorIndex := rand.Intn(len(population))
		if fitnesses[competitorIndex] < bestFitness {
			best, bestFitness = population[competitorIndex], fitnesses[competitorIndex]
		}
	}
	return best
}

func indexOfSoundType(soundType string) int {
	for i, s := range soundTypes {
		if s == soundType {
			return i
		}
	}
	return 0
}

func main() {
	player = playsample.NewPlayer()
	if !player.Initialized {
		log.Fatalln("Error: Audio Player failed to initialize.")
	}
	defer player.Close()

	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandomKick(nil, sampleRate, bitDepth, channels)
		padSoundTypes[i] = "kick"
	}

	activePadIndex = 0

	wnd := g.NewMasterWindow(versionString, 780, 660, g.MasterWindowFlagsNotResizable)
	wnd.Run(loop)
}
