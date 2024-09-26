package synth

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
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

// Settings holds the configuration for generating a sound
type Settings struct {
	SampleRate                 int
	BitDepth                   int
	Channels                   int
	Output                     io.WriteSeeker
	StartFreq                  float64
	EndFreq                    float64
	Duration                   float64
	WaveformType               int
	NoiseAmount                float64
	Attack                     float64
	Decay                      float64
	Sustain                    float64
	Release                    float64
	Drive                      float64
	FilterCutoff               float64
	FilterResonance            float64
	Sweep                      float64
	PitchDecay                 float64
	NumOscillators             int
	OscillatorLevels           []float64
	SaturatorAmount            float64
	FilterBands                []float64
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
func SchroederReverb(samples []float64, decayFactor float64, combDelays []int, allPassDelays []int) ([]float64, error) {
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
