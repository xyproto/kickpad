package main

import (
	"fmt"
	"image/color"
	"math"
	"math/cmplx"
	"math/rand"
	"os"
	"path/filepath"
	"sync"

	g "github.com/AllenDang/giu"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
	"github.com/xyproto/synth"
)

const (
	versionString   = "Kickpad 1.2.0"
	buttonSize      = 100
	numPads         = 16
	maxGenerations  = 1000
	maxStagnation   = 10 // Stop after 10 generations with no fitness improvement
	defaultBitDepth = 16
	populationSize  = 100
	tournamentSize  = 5
	eliteCount      = 10
	mutationRate    = 0.05

	// Minimum and Maximum values for ADSR parameters
	minAttack = 0.05 // seconds
	maxAttack = 0.5  // seconds

	minDecay = 0.05 // seconds
	maxDecay = 0.5  // seconds

	minSustain = 0.1 // Sustain level (0.0 to 1.0)
	maxSustain = 1.0 // Sustain level

	minRelease = 0.05 // seconds
	maxRelease = 1.0  // seconds

	// Additional parameters
	minDrive = 0.0
	maxDrive = 1.0

	minFilterCutoff = 500   // Hz
	maxFilterCutoff = 10000 // Hz

	minSweep = 0.1 // seconds
	maxSweep = 2.0 // seconds

	minPitchDecay = 0.1 // seconds
	maxPitchDecay = 1.5 // seconds

	minNoiseAmount = 0.0
	maxNoiseAmount = 1.0

	minSampleDuration = 0.1 // seconds
	maxSampleDuration = 2.0 // seconds
)

var (
	// List of available sound types for the drop-down
	soundTypes                   = []string{"kick", "clap", "snare", "closed_hh", "open_hh", "rimshot", "tom", "percussion", "ride", "crash", "bass", "xylophone", "lead"}
	soundTypeSelectedIndex int32 = 0 // Index for the selected sound type

	activePadIndex  int
	pads            [numPads]*synth.Settings
	loadedWaveform  []float64                 // Loaded .wav file waveform data
	trainingOngoing bool                      // Indicates whether the GA is running
	wavFilePath     string    = "kick909.wav" // Default .wav file path
	statusMessage   string                    // Status message displayed at the bottom
	cancelTraining  chan bool                 // Channel to cancel GA training
	sampleRates               = []int{44100, 48000, 96000, 192000}
	bitDepth        int       = defaultBitDepth
	sampleRate      int       = sampleRates[0]

	mu sync.Mutex

	// Dropdown selection index for the waveform
	waveformSelectedIndex int32
	sampleRateIndex       int32
	bitDepthSelected      bool
)

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

// compareWaveformsSafe compares two waveforms using both time-domain and frequency-domain MSE
func compareWaveformsSafe(individual *synth.Settings) float64 {
	generatedWaveform, err := individual.Generate(soundTypes[soundTypeSelectedIndex])

	if err != nil {
		return math.Inf(1) // Assign worst fitness if generation fails
	}

	// Ensure both waveforms have the same sample rate
	if individual.SampleRate != sampleRate {
		// Resample if necessary (implement resampling if your application requires it)
		// For simplicity, we assume the sample rates match
	}

	// Calculate time-domain MSE
	timeMSE := compareWaveforms(generatedWaveform, loadedWaveform)

	// Calculate frequency-domain MSE
	freqMSE := compareWaveformsFFT(generatedWaveform, loadedWaveform, sampleRate)

	// Combine the two MSEs with weighting factors
	// You can adjust the weights based on which aspect you want to prioritize
	combinedMSE := 0.5*timeMSE + 0.5*freqMSE

	// Calculate expected duration based on ADSR parameters
	expectedDuration := individual.Attack + individual.Decay + individual.Release // Assuming Sustain does not add to duration

	// Apply penalty if duration is below the minimum threshold
	if expectedDuration < minSampleDuration {
		penalty := (minSampleDuration - expectedDuration) * 1000 // Scale penalty appropriately
		combinedMSE += penalty
	}

	// Apply penalty if duration is above the maximum threshold
	if expectedDuration > maxSampleDuration {
		penalty := (expectedDuration - maxSampleDuration) * 1000 // Scale penalty appropriately
		combinedMSE += penalty
	}

	return combinedMSE
}

// compareWaveformsFFT compares two waveforms in the frequency domain using FFT-based MSE
func compareWaveformsFFT(waveform1, waveform2 []float64, sampleRate int) float64 {
	if waveform1 == nil || waveform2 == nil {
		return math.Inf(1) // Assign worst fitness if any waveform is nil
	}

	// Determine the length for FFT (use the next power of two for efficiency)
	n := nextPowerOfTwo(min(len(waveform1), len(waveform2)))

	// Zero-pad the waveforms to length n
	padded1 := make([]float64, n)
	padded2 := make([]float64, n)
	copy(padded1, waveform1)
	copy(padded2, waveform2)

	// Perform FFT on both waveforms
	complex1 := fft.FFTReal(padded1)
	complex2 := fft.FFTReal(padded2)

	// Compute magnitude spectra
	mag1 := make([]float64, n)
	mag2 := make([]float64, n)
	for i := 0; i < n; i++ {
		mag1[i] = cmplx.Abs(complex1[i])
		mag2[i] = cmplx.Abs(complex2[i])
	}

	// Compute MSE between magnitude spectra
	mse := 0.0
	for i := 0; i < n; i++ {
		diff := mag1[i] - mag2[i]
		mse += diff * diff
	}
	mse /= float64(n)

	return mse
}

// Helper function to find the next power of two greater than or equal to n
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

// Helper function to find the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Compare two waveforms using Mean Squared Error (MSE)
func compareWaveforms(waveform1, waveform2 []float64) float64 {
	if waveform1 == nil || waveform2 == nil {
		return math.Inf(1) // Assign worst fitness if any waveform is nil
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

// Randomize all pads (instead of mutating)
func randomizeAllPads() {
	for i := 0; i < numPads; i++ {
		pads[i] = synth.NewRandom(nil, sampleRate, bitDepth)
	}
}

// Tournament Selection: Selects the best individual from a random subset of the population
func tournamentSelection(population []*synth.Settings, fitnesses []float64, tournamentSize int) *synth.Settings {
	// Randomly select the first competitor
	bestIndex := rand.Intn(len(population))
	best := population[bestIndex]
	bestFitness := fitnesses[bestIndex]

	// Iterate through the rest of the tournament competitors
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

// Single-Point Crossover: Combines two parents to produce two offspring
func singlePointCrossover(parent1, parent2 *synth.Settings) (*synth.Settings, *synth.Settings) {
	child1 := synth.CopySettings(parent1)
	child2 := synth.CopySettings(parent2)

	// Define crossover points for each parameter
	// For simplicity, we'll randomly decide for each parameter whether to take from parent1 or parent2
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
	statusMessage = msg
}

func optimizeSettings(allWaveforms bool) {
	if len(loadedWaveform) == 0 {
		setStatusMessage("Error: No .wav file loaded. Please load a .wav file first.")
		return
	}

	// Display initial status message before starting the training
	setStatusMessage("Training started...")
	g.Update()

	// Initialize population
	population := make([]*synth.Settings, populationSize)
	for i := 0; i < populationSize; i++ {
		population[i] = synth.NewRandom(nil, sampleRate, bitDepth)
		// Ensure WaveformType is within the desired range
		if !allWaveforms {
			population[i].WaveformType = rand.Intn(2) // Sine or Triangle
		} else {
			population[i].WaveformType = rand.Intn(7) // "Sine", "Triangle", "Sawtooth", "Square", "Noise White", "Noise Pink" or "Noise Brown"
		}

		// Clamp ADSR and other parameters to their min and max values
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

		// Evaluate fitness for the current population
		fitnesses := make([]float64, populationSize)
		for i, individual := range population {
			fitnesses[i] = compareWaveformsSafe(individual)
		}

		// Find the best individual in the current population
		currentBestFitness := math.Inf(1)
		currentBestIndex := -1
		for i, fitness := range fitnesses {
			if fitness < currentBestFitness {
				currentBestFitness = fitness
				currentBestIndex = i
			}
		}

		// Update best settings if a better individual is found
		if currentBestIndex != -1 && fitnesses[currentBestIndex] < bestFitness {
			bestFitness = fitnesses[currentBestIndex]
			bestSettings = synth.CopySettings(population[currentBestIndex])
			pads[activePadIndex] = bestSettings // Save the best result to the active pad
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

		// Create a new population
		newPopulation := make([]*synth.Settings, 0, populationSize)

		// Elitism: carry over the best individuals to the new population
		for i := 0; i < eliteCount && i < populationSize; i++ {
			newPopulation = append(newPopulation, synth.CopySettings(bestSettings))
		}

		// Generate the rest of the population through selection, crossover, and mutation
		for len(newPopulation) < populationSize {
			// Selection
			parent1 := tournamentSelection(population, fitnesses, tournamentSize)
			parent2 := tournamentSelection(population, fitnesses, tournamentSize)

			// Crossover
			child1, child2 := singlePointCrossover(parent1, parent2)

			// Mutation
			mutateSettings(child1, false)
			mutateSettings(child2, false)

			// Ensure mutated children respect parameter constraints
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

		// If the new population exceeds the desired size, trim it
		if len(newPopulation) > populationSize {
			newPopulation = newPopulation[:populationSize]
		}

		population = newPopulation

		// Update the status message
		setStatusMessage(fmt.Sprintf("Generation %d: Best fitness = %f", generation, bestFitness))
		g.Update() // Update UI with the new generation status
	}

	// After training, save the best result to the active pad and update the sliders
	pads[activePadIndex] = bestSettings
	g.Update() // Update the sliders with the best result
}

// Function to mutate a single config
func mutateSettings(cfg *synth.Settings, allWaveforms bool) {
	// Mutate the configuration with a certain probability
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

	// Mutate the waveform type with a certain probability
	if rand.Float64() < mutationRate {
		if !allWaveforms {
			cfg.WaveformType = rand.Intn(2) // Sine or Triangle
		} else {
			cfg.WaveformType = rand.Intn(7) // "Sine", "Triangle", "Sawtooth", "Square", "Noise White", "Noise Pink" or "Noise Brown"
		}
	}

	// Mutate the noise amount with a certain probability
	if rand.Float64() < mutationRate {
		cfg.NoiseAmount = clamp(cfg.NoiseAmount*(0.8+rand.Float64()*0.4), minNoiseAmount, maxNoiseAmount)
	}
}

// Helper function to clamp a value between min and max
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// UI functions and pad widget handling
func createPadWidget(cfg *synth.Settings, padLabel string, padIndex int) g.Widget {
	buttonColor := cfg.Color()
	padBorderColor := color.RGBA{0x0, 0x0, 0x0, 0xff} // black
	padTextColor := color.RGBA{0x0, 0x0, 0x0, 0xff}   // black

	luminance := 0.299*float64(buttonColor.R) + 0.587*float64(buttonColor.G) + 0.114*float64(buttonColor.B)

	if luminance < 128 { // dark background color
		padTextColor = color.RGBA{0xff, 0xff, 0xff, 0xff} // white
	}

	if padIndex == activePadIndex {
		padBorderColor = color.RGBA{0xff, 0x0, 0x0, 0xff} // red
	}

	return g.Style().SetColor(g.StyleColorButton, buttonColor).To(
		g.Column(
			g.Style().SetColor(g.StyleColorText, padTextColor).SetColor(g.StyleColorBorder, padBorderColor).To(
				g.Button(padLabel).Size(buttonSize, buttonSize).OnClick(func() {
					// Clear the status message when a pad is clicked
					statusMessage = ""
					// Set the clicked pad as active
					activePadIndex = padIndex
					// Then generate and play the sample (even during training)
					go func() {
						err := synth.FFGeneratePlay(soundTypes[soundTypeSelectedIndex], pads[activePadIndex])
						if err != nil {
							statusMessage = fmt.Sprintf("Error: Failed to generate and play sound: %v", err)
						}
					}()
				})),
			// Mutate button: Mutate the selected pad and update the sliders
			g.Button("Mutate").OnClick(func() {
				mutateSettings(pads[padIndex], true) // Mutate the selected pad and update settings
				activePadIndex = padIndex
				g.Update() // Update the sliders with mutated settings
			}),
			// Save button: Save the current pad's configuration as a .wav file
			g.Button("Save").OnClick(func() {
				pads[padIndex].SampleRate = sampleRate
				pads[padIndex].BitDepth = bitDepth
				fileName, err := pads[padIndex].GenerateAndSaveTo(soundTypes[soundTypeSelectedIndex], ".") // Save the active pad's settings to a .wav file
				if err != nil {
					statusMessage = fmt.Sprintf("Error: Failed to generate and save sound to %s", ".")
				} else {
					statusMessage = fmt.Sprintf("sound saved to %s", fileName)
				}
				g.Update() // Update the status message
			}),
		),
	)
}

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
		g.Label(fmt.Sprintf("Pad %d settings:", activePadIndex+1)),
		g.Dummy(30, 0),
		// Dropdown for sound type selection
		g.Row(
			g.Label("Sound Type"),
			g.Combo("Sound Type", soundTypes[soundTypeSelectedIndex], soundTypes, &soundTypeSelectedIndex).Size(150),
		),
		g.Dummy(30, 0),
		// Sliders for sound parameters
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
				statusMessage = ""
				err := synth.FFGeneratePlay(soundTypes[soundTypeSelectedIndex], pads[activePadIndex])
				if err != nil {
					statusMessage = fmt.Sprintf("Error: Failed to play %s.", soundTypes[soundTypeSelectedIndex])
				}
				g.Update() // Update the status message
			}),
			g.Button("Randomize all").OnClick(func() {
				randomizeAllPads() // Randomize all pads
				g.Update()         // Refresh the UI with randomized settings
			}),
			g.Button("Save").OnClick(func() {
				pads[activePadIndex].SampleRate = sampleRate
				pads[activePadIndex].BitDepth = bitDepth
				fileName, err := pads[activePadIndex].GenerateAndSaveTo(soundTypes[soundTypeSelectedIndex], ".")
				if err != nil {
					statusMessage = fmt.Sprintf("Error: Failed to save %s to %s", soundTypes[soundTypeSelectedIndex], fileName)
				} else {
					statusMessage = fmt.Sprintf("%s saved to %s", soundTypes[soundTypeSelectedIndex], fileName)
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
					g.InputText(&wavFilePath).Size(200),
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

// Conditionally generate the "Find sound similar to WAV" and "Stop training" buttons
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
			g.Button("Find sound similar to WAV").OnClick(func() {
				if !trainingOngoing {
					cancelTraining = make(chan bool)
					trainingOngoing = true
					const allWaveforms = true         // Only Sine and Triangle, or all available waveforms?
					go optimizeSettings(allWaveforms) // Run the optimization in a goroutine
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
