package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/cmplx"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"

	g "github.com/AllenDang/giu"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
	"github.com/xyproto/playsample"
	"github.com/xyproto/synth"
)

const (
	versionString     = "Kickpad 1.5.4"
	channels          = 1
	buttonSize        = 100
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
	//go:embed kick909.wav
	kick909Wav []byte

	soundTypes            = []synth.SoundType{synth.Kick, synth.Clap, synth.Snare, synth.ClosedHH, synth.OpenHH, synth.Rimshot, synth.Tom, synth.Percussion, synth.Ride, synth.Crash, synth.Bass, synth.Xylophone, synth.Lead}
	activePadIndex        int
	pads                  [numPads]*synth.Settings
	padSoundTypes         = make([]synth.SoundType, numPads)
	loadedWaveform        []float64
	trainingOngoing       int32
	wavFilePath           string
	statusMessage         string
	cancelTraining        chan struct{}
	sampleRates               = []int{44100, 48000, 96000, 192000}
	bitDepth              int = defaultBitDepth
	sampleRate            int = sampleRates[0]
	mu                    sync.Mutex
	waveformSelectedIndex int32
	sampleRateIndex       int32
	bitDepthSelected      bool
	player                *playsample.Player
	muPlayer              sync.Mutex
)

func loadWavData(data []byte) error {
	reader := bytes.NewReader(data)
	decoder := wav.NewDecoder(reader)
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		setStatusMessage("Error: Failed to decode embedded .wav data")
		return err
	}
	loadedWaveform = make([]float64, len(buffer.Data))
	for i, sample := range buffer.Data {
		loadedWaveform[i] = float64(sample) / math.MaxInt16
	}
	setStatusMessage("Loaded embedded .wav data")
	return nil
}

func loadWavFile() error {
	filePath := wavFilePath
	if filePath == "" {
		setStatusMessage("No .wav file path provided")
		return errors.New("no .wav file path provided")
	}
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
		loadedWaveform[i] = float64(sample) / math.MaxInt16
	}
	setStatusMessage(fmt.Sprintf("Loaded .wav file: %s", filePath))
	return nil
}

func playLoadedWaveform() error {
	muPlayer.Lock()
	defer muPlayer.Unlock()
	if player == nil || !player.Initialized {
		return errors.New("audio player is not initialized")
	}
	if loadedWaveform == nil || len(loadedWaveform) == 0 {
		return errors.New("no waveform loaded")
	}
	return player.PlayWaveform(loadedWaveform, 44100, 16, 1)
}

func compareWaveformsSafe(individual *synth.Settings) float64 {
	generatedWaveform, err := individual.Generate()
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
	mag1 := make([]float64, n)
	mag2 := make([]float64, n)
	for i := 0; i < n; i++ {
		mag1[i] = cmplx.Abs(complex1[i])
		mag2[i] = cmplx.Abs(complex2[i])
	}
	mse := 0.0
	for i := 0; i < n; i++ {
		diff := mag1[i] - mag2[i]
		mse += diff * diff
	}
	mse /= float64(n)
	return mse
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

func compareWaveforms(waveform1, waveform2 []float64) float64 {
	if waveform1 == nil || waveform2 == nil {
		return math.Inf(1)
	}
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
		var randomSoundType synth.SoundType = synth.Kick
		if rand.Float64() < 0.5 {
			randomSoundType = synth.Snare
		}
		pads[i] = synth.NewRandom(randomSoundType, nil, sampleRate, bitDepth, channels)
	}
}

func tournamentSelection(population []*synth.Settings, fitnesses []float64, tournamentSize int) *synth.Settings {
	bestIndex := rand.Intn(len(population))
	best := population[bestIndex]
	bestFitness := fitnesses[bestIndex]
	for i := 1; i < tournamentSize; i++ {
		competitorIndex := rand.Intn(len(population))
		competitorFitness := fitnesses[competitorIndex]
		if competitorFitness < bestFitness {
			best = population[competitorIndex]
			bestFitness = competitorFitness
			bestIndex = competitorIndex
		}
	}
	return best
}

func singlePointCrossover(parent1, parent2 *synth.Settings) (*synth.Settings, *synth.Settings) {
	child1 := synth.CopySettings(parent1)
	child2 := synth.CopySettings(parent2)
	if rand.Float64() < 0.5 {
		child1.Attack = parent2.Attack
		child2.Attack = parent1.Attack
	}
	if rand.Float64() < 0.5 {
		child1.Decay = parent2.Decay
		child2.Decay = parent1.Decay
	}
	if rand.Float64() < 0.5 {
		child1.Sustain = parent2.Sustain
		child2.Sustain = parent1.Sustain
	}
	if rand.Float64() < 0.5 {
		child1.Release = parent2.Release
		child2.Release = parent1.Release
	}
	if rand.Float64() < 0.5 {
		child1.Drive = parent2.Drive
		child2.Drive = parent1.Drive
	}
	if rand.Float64() < 0.5 {
		child1.FilterCutoff = parent2.FilterCutoff
		child2.FilterCutoff = parent1.FilterCutoff
	}
	if rand.Float64() < 0.5 {
		child1.Sweep = parent2.Sweep
		child2.Sweep = parent1.Sweep
	}
	if rand.Float64() < 0.5 {
		child1.PitchDecay = parent2.PitchDecay
		child2.PitchDecay = parent1.PitchDecay
	}
	if rand.Float64() < 0.5 {
		child1.WaveformType = parent2.WaveformType
		child2.WaveformType = parent1.WaveformType
	}
	if rand.Float64() < 0.5 {
		child1.NoiseAmount = parent2.NoiseAmount
		child2.NoiseAmount = parent1.NoiseAmount
	}
	return child1, child2
}

func setStatusMessage(msg string) {
	mu.Lock()
	defer mu.Unlock()
	statusMessage = msg
}

func optimizeSettings(allWaveforms bool) {
	if len(loadedWaveform) == 0 {
		setStatusMessage("Error: No .wav file loaded. Please load a .wav file first.")
		return
	}
	setStatusMessage("Training started...")
	// Initialize population
	population := make([]*synth.Settings, populationSize)
	for i := 0; i < populationSize; i++ {
		population[i] = synth.NewRandom(synth.Kick, nil, sampleRate, bitDepth, channels)
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
	for generation := 0; generation < maxGenerations && atomic.LoadInt32(&trainingOngoing) == 1; generation++ {
		select {
		case <-cancelTraining:
			setStatusMessage("Training canceled.")
			atomic.StoreInt32(&trainingOngoing, 0)
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
				atomic.StoreInt32(&trainingOngoing, 0)
				return
			}
		} else {
			stagnationCount++
			if stagnationCount >= maxStagnation {
				setStatusMessage(fmt.Sprintf("Training stopped due to no improvement in %d generations.", maxStagnation))
				atomic.StoreInt32(&trainingOngoing, 0)
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
			mutateSettings(child1, false)
			mutateSettings(child2, false)
			child1.Attack = clamp(child1.Attack, minAttack, maxAttack)
			child1.Decay = clamp(child1.Decay, minDecay, maxDecay)
			child1.Sustain = clamp(child1.Sustain, minSustain, maxSustain)
			child1.Release = clamp(child1.Release, minRelease, maxRelease)
			child1.Drive = clamp(child1.Drive, minDrive, maxDrive)
			child1.FilterCutoff = clamp(child1.FilterCutoff, minFilterCutoff, maxFilterCutoff)
			child1.Sweep = clamp(child1.Sweep, minSweep, maxSweep)
			child1.PitchDecay = clamp(child1.PitchDecay, minPitchDecay, maxPitchDecay)
			child1.NoiseAmount = clamp(child1.NoiseAmount, minNoiseAmount, maxNoiseAmount)
			child2.Attack = clamp(child2.Attack, minAttack, maxAttack)
			child2.Decay = clamp(child2.Decay, minDecay, maxDecay)
			child2.Sustain = clamp(child2.Sustain, minSustain, maxSustain)
			child2.Release = clamp(child2.Release, minRelease, maxRelease)
			child2.Drive = clamp(child2.Drive, minDrive, maxDrive)
			child2.FilterCutoff = clamp(child2.FilterCutoff, minFilterCutoff, maxFilterCutoff)
			child2.Sweep = clamp(child2.Sweep, minSweep, maxSweep)
			child2.PitchDecay = clamp(child2.PitchDecay, minPitchDecay, maxPitchDecay)
			child2.NoiseAmount = clamp(child2.NoiseAmount, minNoiseAmount, maxNoiseAmount)
			newPopulation = append(newPopulation, child1, child2)
		}
		if len(newPopulation) > populationSize {
			newPopulation = newPopulation[:populationSize]
		}
		population = newPopulation
		setStatusMessage(fmt.Sprintf("Generation %d: Best fitness = %f", generation, bestFitness))
	}
	pads[activePadIndex] = bestSettings
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
				g.Button(padLabel).Size(buttonSize, buttonSize).OnClick(func() {
					activePadIndex = padIndex
					setStatusMessage("")
					go func() {
						if err := GeneratePlay(pads[activePadIndex]); err != nil {
							setStatusMessage(fmt.Sprintf("Error: Failed to play sound: %v", err))
						} else {
							setStatusMessage(fmt.Sprintf("Playing sound from %s", padLabel))
						}
					}()
				}),
			),
		),
	)
}

func createSlidersForSelectedPad() g.Widget {
	cfg := pads[activePadIndex]
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

	var soundTypeStrings []string
	for _, soundType := range soundTypes {
		soundTypeStrings = append(soundTypeStrings, soundType.String())
	}

	soundTypeSelectedIndex := int32(pads[activePadIndex].SoundType)

	return g.Column(
		g.Label(fmt.Sprintf("Pad %d settings:", activePadIndex+1)),
		g.Dummy(30, 0),
		g.Row(
			g.Label("Sound Type"),
			g.Combo("Sound Type", pads[activePadIndex].SoundType.String(), soundTypeStrings, &soundTypeSelectedIndex).Size(150).OnChange(func() {
				pads[activePadIndex] = synth.NewRandom(soundTypes[soundTypeSelectedIndex], nil, sampleRate, bitDepth, channels)
				pads[activePadIndex].SoundType = soundTypes[soundTypeSelectedIndex]
			}),
		),
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
				err := GeneratePlay(pads[activePadIndex])
				if err != nil {
					setStatusMessage(fmt.Sprintf("Error: Failed to play %s.", padSoundTypes[activePadIndex]))
				}
			}),
			g.Button("Randomize").OnClick(func() {
				var randomSoundType synth.SoundType = synth.Kick
				if rand.Float64() < 0.5 {
					randomSoundType = synth.Snare
				}
				pads[activePadIndex] = synth.NewRandom(randomSoundType, nil, sampleRate, bitDepth, channels)
			}),
			g.Button("Randomize all").OnClick(func() {
				randomizeAllPads()
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
					g.Button("Save").OnClick(func() {
						pads[activePadIndex].SampleRate = sampleRate
						pads[activePadIndex].BitDepth = bitDepth
						fileName, err := pads[activePadIndex].GenerateAndSaveTo(".")
						if err != nil {
							setStatusMessage(fmt.Sprintf("Error: Failed to save %s to %s", padSoundTypes[activePadIndex], fileName))
						} else {
							os.Rename(fileName, wavFilePath)
							fileName = wavFilePath
							setStatusMessage(fmt.Sprintf("%s saved to %s", padSoundTypes[activePadIndex], fileName))
						}
					}),
				),
				g.Condition(len(loadedWaveform) > 0 || atomic.LoadInt32(&trainingOngoing) == 1,
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
		if atomic.LoadInt32(&trainingOngoing) == 1 {
			return g.Row(
				g.Button("Stop training").OnClick(func() {
					if atomic.LoadInt32(&trainingOngoing) == 1 {
						close(cancelTraining)
					}
				}),
				g.Button("Play WAV").OnClick(func() {
					err := playLoadedWaveform()
					if err != nil {
						setStatusMessage("Error: Failed to play WAV")
					}
				}),
			)
		}
		return g.Row(
			g.Button("Find sound similar to WAV").OnClick(func() {
				if atomic.LoadInt32(&trainingOngoing) == 0 {
					cancelTraining = make(chan struct{})
					atomic.StoreInt32(&trainingOngoing, 1)
					const allWaveforms = true
					go optimizeSettings(allWaveforms)
				}
			}),
			g.Button("Play WAV").OnClick(func() {
				err := playLoadedWaveform()
				if err != nil {
					setStatusMessage("Error: Failed to play WAV")
				}
			}),
		)
	}
	return g.Dummy(0, 0)
}

func GeneratePlay(cfg *synth.Settings) error {
	muPlayer.Lock()
	defer muPlayer.Unlock()
	if player == nil || !player.Initialized {
		return errors.New("audio player is not initialized")
	}
	samples, err := cfg.Generate()
	if err != nil {
		return err
	}
	return player.PlayWaveform(samples, cfg.SampleRate, cfg.BitDepth, cfg.Channels)
}

func main() {
	player = playsample.NewPlayer()
	if !player.Initialized {
		log.Fatalln("Error: Audio Player failed to initialize.")
	}
	defer player.Close()
	err := loadWavData(kick909Wav)
	if err != nil {
		log.Fatalln("Error loading embedded .wav data:", err)
	}
	const defaultSoundType = synth.Kick
	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandom(defaultSoundType, nil, sampleRate, bitDepth, channels)
	}
	activePadIndex = 0
	setStatusMessage(versionString)
	g.NewMasterWindow(versionString, 780, 445, g.MasterWindowFlagsNotResizable).Run(loop)
}
