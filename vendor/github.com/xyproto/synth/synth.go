package synth

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"math/rand"
)

// Constants for waveform types
const (
	WaveSine = iota
	WaveTriangle
	WaveSawtooth
	WaveSquare
	WaveWhiteNoise
	WavePinkNoise
	WaveBrownNoise
)

// Constants for noise types
const (
	NoiseNone = iota
	NoiseWhite
	NoisePink
	NoiseBrown
)

// Settings holds the configuration for generating a sound
type Settings struct {
	StartFreq                  float64
	EndFreq                    float64
	SampleRate                 int
	Duration                   float64
	WaveformType               int
	Attack                     float64
	Decay                      float64
	Sustain                    float64
	Release                    float64
	Drive                      float64
	FilterCutoff               float64
	FilterResonance            float64
	Sweep                      float64
	PitchDecay                 float64
	NoiseType                  int
	NoiseAmount                float64
	Output                     io.WriteSeeker
	NumOscillators             int
	OscillatorLevels           []float64
	SaturatorAmount            float64
	FilterBands                []float64
	BitDepth                   int
	FadeDuration               float64
	SmoothFrequencyTransitions bool
}

// FadeCurve defines a type for fade curve functions
type FadeCurve func(t float64) float64

// SawtoothOscillator generates a sawtooth waveform at a specific frequency
func SawtoothOscillator(freq float64, length int, sampleRate int) []float64 {
	osc := make([]float64, length)
	for i := range osc {
		osc[i] = 2 * (float64(i)*freq/float64(sampleRate) - math.Floor(0.5+float64(i)*freq/float64(sampleRate)))
	}
	return osc
}

// DetunedOscillators generates multiple detuned sawtooth oscillators and combines them
func DetunedOscillators(freq float64, detune []float64, length int, sampleRate int) []float64 {
	numOsc := len(detune)
	combined := make([]float64, length)
	for _, d := range detune {
		osc := SawtoothOscillator(freq*(1+d), length, sampleRate)
		for i := range combined {
			combined[i] += osc[i] / float64(numOsc)
		}
	}
	return combined
}

// ApplyEnvelope applies an ADSR envelope to the waveform
func ApplyEnvelope(samples []float64, attack, decay, sustain, release float64, sampleRate int) []float64 {
	adsr := make([]float64, len(samples))
	totalDuration := float64(len(samples)) / float64(sampleRate)
	for i := range samples {
		t := float64(i) / float64(sampleRate)
		var envelope float64
		if t < attack {
			envelope = t / attack
		} else if t < attack+decay {
			envelope = 1 - (t-attack)/decay*(1-sustain)
		} else if t < totalDuration-release {
			envelope = sustain
		} else if t < totalDuration {
			envelope = sustain * (1 - (t-(totalDuration-release))/release)
		} else {
			envelope = 0
		}
		adsr[i] = samples[i] * envelope
	}
	return adsr
}

// LowPassFilter applies a basic low-pass filter to the samples
func LowPassFilter(samples []float64, cutoff float64, sampleRate int) []float64 {
	filtered := make([]float64, len(samples))
	rc := 1.0 / (2.0 * math.Pi * cutoff)
	dt := 1.0 / float64(sampleRate)
	alpha := dt / (rc + dt)

	if len(samples) == 0 {
		return samples
	}

	prev := samples[0]       // Initialize with the first sample
	filtered[0] = samples[0] // The first sample remains the same

	for i := 1; i < len(samples); i++ {
		filtered[i] = alpha*samples[i] + (1-alpha)*prev
		prev = filtered[i]
	}
	return filtered
}

// Drive applies a simple drive effect by scaling and clipping
func Drive(samples []float64, gain float64) []float64 {
	driven := make([]float64, len(samples))
	for i, sample := range samples {
		driven[i] = sample * gain
		if driven[i] > 1 {
			driven[i] = 1
		} else if driven[i] < -1 {
			driven[i] = -1
		}
	}
	return driven
}

// Limiter ensures the signal doesn't exceed [-1, 1] range
func Limiter(samples []float64) []float64 {
	limited := make([]float64, len(samples))
	for i, sample := range samples {
		if sample > 1 {
			limited[i] = 1
		} else if sample < -1 {
			limited[i] = -1
		} else {
			limited[i] = sample
		}
	}
	return limited
}

// GenerateKick generates the kick drum sound based on the settings
func (cfg *Settings) GenerateKick() error {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate waveform based on the WaveformType
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration)
		var sample float64

		switch cfg.WaveformType {
		case WaveSine:
			sample = math.Sin(2 * math.Pi * frequency * t)
		case WaveTriangle:
			sample = 2*math.Abs(2*(t*frequency-math.Floor(t*frequency+0.5))) - 1
		case WaveSawtooth:
			sample = 2 * (t*frequency - math.Floor(0.5+t*frequency))
		case WaveSquare:
			sample = math.Copysign(1.0, math.Sin(2*math.Pi*frequency*t))
		case WaveWhiteNoise:
			sample = (rand.Float64()*2 - 1) * cfg.NoiseAmount
		case WavePinkNoise:
			sample = GenerateNoise(NoisePink, 1, cfg.NoiseAmount)[0]
		case WaveBrownNoise:
			sample = GenerateNoise(NoiseBrown, 1, cfg.NoiseAmount)[0]
		default:
			return fmt.Errorf("unsupported waveform type: %d", cfg.WaveformType)
		}

		if len(cfg.OscillatorLevels) > 0 {
			sample *= cfg.OscillatorLevels[0] // Apply the first oscillator level
		}

		// Apply envelope (ADSR)
		sample *= cfg.ApplyEnvelopeAtTime(t)

		// Apply drive (distortion)
		sample = cfg.ApplyDrive(sample)

		samples[i] = sample
	}

	// Apply limiter to the samples
	samples = Limiter(samples)

	// Write to output
	if cfg.Output != nil {
		return SaveToWav(cfg.Output, samples, cfg.SampleRate, cfg.BitDepth)
	}

	// If no output is specified, return an error
	return errors.New("no output specified for GenerateKick")
}

// ApplyEnvelopeAtTime generates the ADSR envelope at a specific time
func (cfg *Settings) ApplyEnvelopeAtTime(t float64) float64 {
	attack, decay, sustain, release := cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release
	duration := cfg.Duration

	if t < attack {
		return t / attack
	}
	if t < attack+decay {
		return 1.0 - (t-attack)/decay*(1.0-sustain)
	}
	if t < duration-release {
		return sustain
	}
	if t < duration {
		return sustain * (1.0 - (t-(duration-release))/release)
	}
	return 0.0
}

// ApplyDrive applies a drive (distortion) effect to the sample
func (cfg *Settings) ApplyDrive(sample float64) float64 {
	if cfg.Drive > 0 {
		return sample * (1 + cfg.Drive) / (1 + cfg.Drive*math.Abs(sample))
	}
	return sample
}

// LinearSummation mixes multiple audio samples by averaging them together.
// It automatically clamps the sum to avoid overflow and distortion.
func LinearSummation(samples ...[]float64) ([]float64, error) {
	if len(samples) == 0 {
		return nil, errors.New("no samples provided")
	}

	numSamples := len(samples[0])
	combined := make([]float64, numSamples)

	// Sum the samples from each input
	for i := 0; i < numSamples; i++ {
		sum := float64(0)
		for _, sample := range samples {
			if len(sample) != numSamples {
				return nil, errors.New("mismatched sample lengths")
			}
			sum += sample[i]
		}

		// Average the sum by dividing by the number of input samples
		average := sum / float64(len(samples))

		// Clamp the result to avoid overflow
		if average > 1 {
			combined[i] = 1
		} else if average < -1 {
			combined[i] = -1
		} else {
			combined[i] = average
		}
	}

	return combined, nil
}

// WeightedSummation mixes multiple audio samples by applying a weight to each sample.
// Each sample's amplitude is scaled by its corresponding weight before summing.
func WeightedSummation(weights []float64, samples ...[]float64) ([]float64, error) {
	if len(weights) != len(samples) {
		return nil, errors.New("number of weights must match number of samples")
	}

	if len(samples) == 0 {
		return nil, errors.New("no samples provided")
	}

	numSamples := len(samples[0])
	combined := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		sum := float64(0)
		for j, sample := range samples {
			if len(sample) != numSamples {
				return nil, errors.New("mismatched sample lengths")
			}
			sum += sample[i] * weights[j]
		}
		// Clamp the result to avoid overflow
		if sum > 1 {
			sum = 1
		} else if sum < -1 {
			sum = -1
		}
		combined[i] = sum
	}

	return combined, nil
}

// RMSMixing mixes audio samples using the Root Mean Square method.
func RMSMixing(samples ...[]float64) ([]float64, error) {
	if len(samples) == 0 {
		return nil, errors.New("no samples provided")
	}

	numSamples := len(samples[0])
	combined := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		sumSquares := float64(0)
		for _, sample := range samples {
			if len(sample) != numSamples {
				return nil, errors.New("mismatched sample lengths")
			}
			// Square the sample value and accumulate
			sumSquares += sample[i] * sample[i]
		}
		// Calculate RMS by taking the square root of the mean of squares
		rms := math.Sqrt(sumSquares / float64(len(samples)))

		// Clamp the result to [-1, 1] range
		if rms > 1 {
			rms = 1
		} else if rms < -1 {
			rms = -1
		}
		combined[i] = rms
	}

	return combined, nil
}

// AnalyzeHighestFrequency estimates the highest frequency in the audio signal
func AnalyzeHighestFrequency(samples []float64, sampleRate int) float64 {
	zeroCrossings := 0
	l := len(samples)
	for i := 1; i < l; i++ {
		if (samples[i-1] > 0 && samples[i] < 0) || (samples[i-1] < 0 && samples[i] > 0) {
			zeroCrossings++
		}
	}
	duration := float64(l-1) / float64(sampleRate)
	if duration == 0 {
		return 0
	}
	frequency := float64(zeroCrossings) / (2 * duration)
	return frequency
}

// NormalizeSamples scales the samples so the peak amplitude matches the given max amplitude
func NormalizeSamples(samples []float64, targetPeak float64) []float64 {
	currentPeak := FindPeakAmplitude(samples)
	if currentPeak == 0 {
		return samples
	}
	scale := targetPeak / currentPeak
	normalizedSamples := make([]float64, len(samples))
	for i, sample := range samples {
		normalized := sample * scale
		// Clamp the values to the [-1, 1] range after scaling
		if normalized > 1 {
			normalizedSamples[i] = 1
		} else if normalized < -1 {
			normalizedSamples[i] = -1
		} else {
			normalizedSamples[i] = normalized
		}
	}
	return normalizedSamples
}

// FindPeakAmplitude returns the maximum absolute amplitude in the sample set
func FindPeakAmplitude(samples []float64) float64 {
	maxAmplitude := float64(0)
	for _, sample := range samples {
		if abs := math.Abs(sample); abs > maxAmplitude {
			maxAmplitude = abs
		}
	}
	return maxAmplitude
}

// PadSamples pads the shorter waveform with zeros to make both waveforms the same length.
func PadSamples(wave1, wave2 []float64) ([]float64, []float64) {
	length1 := len(wave1)
	length2 := len(wave2)

	if length1 == length2 {
		return wave1, wave2
	}

	// Pad the shorter waveform with zeros
	if length1 < length2 {
		paddedWave1 := make([]float64, length2)
		copy(paddedWave1, wave1)
		return paddedWave1, wave2
	}

	paddedWave2 := make([]float64, length1)
	copy(paddedWave2, wave2)
	return wave1, paddedWave2
}

// CopySettings creates a deep copy of a Settings struct
func CopySettings(cfg *Settings) *Settings {
	newCfg := *cfg
	newCfg.OscillatorLevels = append([]float64(nil), cfg.OscillatorLevels...) // Deep copy the slice
	return &newCfg
}

// GenerateKickWaveform generates the kick waveform and returns it as a slice of float64 samples (without writing to disk).
func (cfg *Settings) GenerateKickWaveform() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration)
		var sample float64

		switch cfg.WaveformType {
		case WaveSine:
			sample = math.Sin(2 * math.Pi * frequency * t)
		case WaveTriangle:
			sample = 2*math.Abs(2*(t*frequency-math.Floor(t*frequency+0.5))) - 1
		case WaveSawtooth:
			sample = 2 * (t*frequency - math.Floor(0.5+t*frequency))
		case WaveSquare:
			sample = math.Copysign(1.0, math.Sin(2*math.Pi*frequency*t))
		case WaveWhiteNoise:
			sample = (rand.Float64()*2 - 1) * cfg.NoiseAmount
		case WavePinkNoise:
			sample = GenerateNoise(NoisePink, 1, cfg.NoiseAmount)[0]
		case WaveBrownNoise:
			sample = GenerateNoise(NoiseBrown, 1, cfg.NoiseAmount)[0]
		default:
			return nil, fmt.Errorf("unsupported waveform type: %d", cfg.WaveformType)
		}

		if len(cfg.OscillatorLevels) > 0 {
			sample *= cfg.OscillatorLevels[0]
		}

		sample *= cfg.ApplyEnvelopeAtTime(t)
		sample = cfg.ApplyDrive(sample)
		samples[i] = sample
	}

	samples = Limiter(samples)
	return samples, nil
}

// Color returns a color that represents the current kick config
func (cfg *Settings) Color() color.RGBA {
	hasher := sha1.New()
	hasher.Write([]byte(fmt.Sprintf("%d%f%f%f%f%f%f%f%f", cfg.WaveformType, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.Drive, cfg.FilterCutoff, cfg.Sweep, cfg.PitchDecay)))
	hashBytes := hasher.Sum(nil)
	// Convert the first few bytes of the hash into an RGB color
	r := hashBytes[0]
	g := hashBytes[1]
	b := hashBytes[2]
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// HighPassFilter applies a basic high-pass filter to the samples
func HighPassFilter(samples []float64, cutoff float64, sampleRate int) []float64 {
	filtered := make([]float64, len(samples))
	rc := 1.0 / (2.0 * math.Pi * cutoff)
	dt := 1.0 / float64(sampleRate)
	alpha := rc / (rc + dt)

	if len(samples) == 0 {
		return samples
	}

	filtered[0] = samples[0]

	for i := 1; i < len(samples); i++ {
		filtered[i] = alpha * (filtered[i-1] + samples[i] - samples[i-1])
	}
	return filtered
}

// BandPassFilter applies a band-pass filter to the samples
func BandPassFilter(samples []float64, lowCutoff, highCutoff float64, sampleRate int) []float64 {
	lowPassed := LowPassFilter(samples, highCutoff, sampleRate)
	return HighPassFilter(lowPassed, lowCutoff, sampleRate)
}

// SchroederReverb applies a high-quality reverb effect using the Schroeder algorithm
func SchroederReverb(samples []float64, sampleRate int, decayFactor float64, combDelays []int, allPassDelays []int) ([]float64, error) {
	if len(combDelays) != 4 || len(allPassDelays) != 2 {
		return nil, errors.New("SchroederReverb expects 4 comb delays and 2 all-pass delays")
	}

	// Create buffers for the comb filters
	combBuffers := make([][]float64, 4)
	for i := range combBuffers {
		combBuffers[i] = make([]float64, combDelays[i])
	}

	// Apply comb filters
	combFiltered := make([]float64, len(samples))
	for i := range samples {
		for j := range combBuffers {
			delayIndex := i % combDelays[j]
			combBuffers[j][delayIndex] = samples[i] + combBuffers[j][delayIndex]*decayFactor
			combFiltered[i] += combBuffers[j][delayIndex]
		}
	}

	// Create buffers for the all-pass filters
	allPassBuffers := make([][]float64, 2)
	for i := range allPassBuffers {
		allPassBuffers[i] = make([]float64, allPassDelays[i])
	}

	// Apply all-pass filters
	reverbOutput := combFiltered
	for j := range allPassBuffers {
		buffer := allPassBuffers[j]
		delay := allPassDelays[j]
		output := make([]float64, len(reverbOutput))
		for i := range reverbOutput {
			delayIndex := i % delay
			bufOut := buffer[delayIndex]
			buffer[delayIndex] = reverbOutput[i] + bufOut*decayFactor
			output[i] = buffer[delayIndex] - bufOut
		}
		reverbOutput = output
	}

	return reverbOutput, nil
}

func ApplyPitchModulation(samples []float64, modFreq, modDepth float64, sampleRate int) []float64 {
	modulated := make([]float64, len(samples))
	for i := 0; i < len(samples); i++ {
		t := float64(i) / float64(sampleRate)
		mod := math.Sin(2*math.Pi*modFreq*t) * modDepth
		value := samples[i] * math.Pow(2, mod)
		// Clamp the value
		if value > 1.0 {
			value = 1.0
		} else if value < -1.0 {
			value = -1.0
		}
		modulated[i] = value
	}
	return modulated
}

// ApplyPanning applies stereo panning to the samples. pan should be in the range [-1, 1], where -1 is full left and 1 is full right.
func ApplyPanning(samples []float64, pan float64) ([]float64, []float64) {
	leftChannel := make([]float64, len(samples))
	rightChannel := make([]float64, len(samples))
	leftGain := math.Cos((pan + 1) * math.Pi / 4) // Equal-power panning
	rightGain := math.Sin((pan + 1) * math.Pi / 4)

	for i := range samples {
		leftChannel[i] = samples[i] * leftGain
		rightChannel[i] = samples[i] * rightGain
	}

	return leftChannel, rightChannel
}

// GenerateNoise generates noise based on the selected noise type
func GenerateNoise(noiseType int, length int, amount float64) []float64 {
	noise := make([]float64, length)
	switch noiseType {
	case NoiseWhite:
		for i := range noise {
			noise[i] = (rand.Float64()*2 - 1) * amount
		}
	case NoisePink:
		// Basic pink noise approximation
		var b0, b1, b2, b3, b4, b5, b6 float64
		for i := range noise {
			white := rand.Float64()*2 - 1
			b0 = 0.99886*b0 + white*0.0555179
			b1 = 0.99332*b1 + white*0.0750759
			b2 = 0.96900*b2 + white*0.1538520
			b3 = 0.86650*b3 + white*0.3104856
			b4 = 0.55000*b4 + white*0.5329522
			b5 = -0.7616*b5 - white*0.0168980
			value := (b0 + b1 + b2 + b3 + b4 + b5 + b6 + white*0.5362) * amount / 3.5
			b6 = white * 0.115926
			// Clamp the value
			if value > amount {
				value = amount
			} else if value < -amount {
				value = -amount
			}
			noise[i] = value
		}
	case NoiseBrown:
		// Brownian noise generation
		var lastOutput float64
		for i := range noise {
			white := (rand.Float64()*2 - 1) * amount / 10
			value := (lastOutput + (0.02 * white)) / 1.02
			lastOutput = value
			value *= 3.5 // (roughly) compensate for gain
			// Clamp the value
			if value > amount {
				value = amount
			} else if value < -amount {
				value = -amount
			}
			noise[i] = value
		}
	}
	return noise
}

// ApplyFrequencyModulation applies frequency modulation to a waveform using a modulator frequency and depth
func ApplyFrequencyModulation(samples []float64, modFreq, modDepth float64, sampleRate int) []float64 {
	modulated := make([]float64, len(samples))
	carrierPhase := 0.0
	modulatorPhase := 0.0
	for i := range samples {
		carrierFreq := modDepth * math.Sin(2*math.Pi*modFreq*modulatorPhase)
		carrierPhase += carrierFreq / float64(sampleRate)
		modulated[i] = math.Sin(2*math.Pi*carrierPhase) * samples[i]
		modulatorPhase += 1.0 / float64(sampleRate)
	}
	return modulated
}

// ApplyFadeIn applies a fade-in to the start of the samples using the specified fade curve
func ApplyFadeIn(samples []float64, fadeDuration float64, sampleRate int, curve FadeCurve) []float64 {
	fadeSamples := int(fadeDuration * float64(sampleRate))
	if fadeSamples > len(samples) {
		fadeSamples = len(samples)
	}
	for i := 0; i < fadeSamples; i++ {
		t := float64(i) / float64(fadeSamples)
		multiplier := curve(t)
		samples[i] *= multiplier
	}
	return samples
}

// ApplyFadeOut applies a fade-out to the end of the samples using the specified fade curve
func ApplyFadeOut(samples []float64, fadeDuration float64, sampleRate int, curve FadeCurve) []float64 {
	totalSamples := len(samples)
	fadeSamples := int(fadeDuration * float64(sampleRate))
	if fadeSamples > totalSamples {
		fadeSamples = totalSamples
	}
	for i := 0; i < fadeSamples; i++ {
		t := float64(i) / float64(fadeSamples)
		multiplier := curve(1.0 - t)
		index := totalSamples - fadeSamples + i
		if index >= 0 && index < totalSamples {
			samples[index] *= multiplier
		}
	}
	return samples
}

// GenerateSweepWaveform generates a frequency sweep waveform based on the settings.
func (cfg *Settings) GenerateSweepWaveform() ([]float64, error) {
	numSamples := int(cfg.Duration * float64(cfg.SampleRate))
	samples := make([]float64, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		// Calculate the frequency at time t
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration)
		var sample float64

		switch cfg.WaveformType {
		case WaveSine:
			sample = math.Sin(2 * math.Pi * frequency * t)
		case WaveTriangle:
			sample = 2*math.Abs(2*(t*frequency-math.Floor(t*frequency+0.5))) - 1
		case WaveSawtooth:
			sample = 2 * (t*frequency - math.Floor(0.5+t*frequency))
		case WaveSquare:
			sample = math.Copysign(1.0, math.Sin(2*math.Pi*frequency*t))
		case WaveWhiteNoise:
			sample = (rand.Float64()*2 - 1) * cfg.NoiseAmount
		case WavePinkNoise:
			sample = GenerateNoise(NoisePink, 1, cfg.NoiseAmount)[0]
		case WaveBrownNoise:
			sample = GenerateNoise(NoiseBrown, 1, cfg.NoiseAmount)[0]
		default:
			return nil, fmt.Errorf("unsupported waveform type: %d", cfg.WaveformType)
		}

		samples[i] = sample
	}

	return samples, nil
}

// ApplyChorus applies a chorus effect to the samples
func ApplyChorus(samples []float64, sampleRate int, delaySec float64, depth float64, rate float64) []float64 {
	delaySamples := int(delaySec * float64(sampleRate))
	modulated := make([]float64, len(samples))
	for i := 0; i < len(samples); i++ {
		t := float64(i) / float64(sampleRate)
		modulation := depth * math.Sin(2*math.Pi*rate*t)
		delay := int(float64(delaySamples) * (1 + modulation))
		index := i - delay
		if index >= 0 && index < len(samples) {
			modulated[i] = (samples[i] + samples[index]) / 2
		} else {
			modulated[i] = samples[i]
		}
	}
	return modulated
}

// LinearFade is a linear fade curve
func LinearFade(t float64) float64 {
	return t
}

// QuadraticFade is a quadratic (ease-in) fade curve
func QuadraticFade(t float64) float64 {
	return t * t
}

// ExponentialFade is an exponential fade curve
func ExponentialFade(t float64) float64 {
	return math.Pow(2, 10*(t-1))
}

// LogarithmicFade is a logarithmic fade curve
func LogarithmicFade(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Log10(t*9 + 1)
}

// SineFade is a sinusoidal fade curve
func SineFade(t float64) float64 {
	return math.Sin((t * math.Pi) / 2)
}
