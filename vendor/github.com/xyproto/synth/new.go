package synth

import (
	"errors"
	"io"
	"math/rand"
)

// NewSettings creates a new Settings instance with default values for a kick drum
func NewSettings(startFreq, endFreq float64, sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	if sampleRate <= 0 || duration <= 0 {
		return nil, errors.New("invalid sample rate or duration")
	}

	return &Settings{
		StartFreq:                  startFreq, // Starting frequency (Hz)
		EndFreq:                    endFreq,   // Ending frequency (Hz)
		SampleRate:                 sampleRate,
		Duration:                   duration,                         // Duration in seconds
		WaveformType:               WaveSine,                         // Default waveform type
		Attack:                     0.005,                            // Attack time in seconds
		Decay:                      0.3,                              // Decay time in seconds
		Sustain:                    0.2,                              // Sustain level
		Release:                    0.3,                              // Release time in seconds
		Drive:                      0.2,                              // Drive (distortion) amount
		FilterCutoff:               5000,                             // Filter cutoff frequency (Hz)
		Sweep:                      0.7,                              // Pitch sweep amount
		PitchDecay:                 0.4,                              // Pitch envelope decay time
		NoiseType:                  NoiseNone,                        // Default to no noise
		NoiseAmount:                0.0,                              // Amount of noise to mix in
		Output:                     output,                           // Output file (WAV)
		NumOscillators:             1,                                // Default to 1 oscillator
		OscillatorLevels:           []float64{1.0},                   // Default oscillator level
		SaturatorAmount:            0.3,                              // Saturation amount
		FilterBands:                []float64{200.0, 1000.0, 3000.0}, // Multi-band filter frequencies
		BitDepth:                   bitDepth,                         // Audio bit depth (16 or 24 bits)
		FadeDuration:               0.01,                             // Fade in/out duration in seconds
		SmoothFrequencyTransitions: true,                             // Enable smooth frequency transitions by default
	}, nil
}

// NewRandom generates random kick drum settings
func NewRandom() *Settings {
	cfg, _ := NewSettings(55.0, 30.0, 96000, 1.0, 16, nil)
	cfg.Attack = rand.Float64() * 0.02
	cfg.Decay = 0.2 + rand.Float64()*0.8
	cfg.Sustain = rand.Float64() * 0.5
	cfg.Release = 0.2 + rand.Float64()*0.5
	cfg.Drive = rand.Float64()
	cfg.FilterCutoff = 2000 + rand.Float64()*6000
	cfg.Sweep = rand.Float64() * 1.5
	cfg.PitchDecay = rand.Float64() * 1.5
	cfg.FadeDuration = rand.Float64() * 0.1
	if rand.Float64() < 0.1 {
		cfg.SmoothFrequencyTransitions = false
	} else {
		cfg.SmoothFrequencyTransitions = true
	}
	if rand.Float64() < 0.1 {
		cfg.WaveformType = rand.Intn(7)
	} else {
		cfg.WaveformType = rand.Intn(2)
	}
	return cfg
}

// NewExperimental creates an experimental kick drum sound
func NewExperimental(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(80.0, 20.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSawtooth       // Sawtooth for a sharper, edgier sound
	cfg.Attack = 0.001                    // Very quick attack
	cfg.Decay = 0.7                       // Longer decay
	cfg.Release = 0.4                     // Extended release
	cfg.Drive = 0.8                       // Strong drive for distortion
	cfg.FilterCutoff = 3000               // Low cutoff for a dark, experimental tone
	cfg.Sweep = 1.2                       // Extreme sweep for dramatic pitch variation
	cfg.PitchDecay = 0.8                  // Heavily exaggerated pitch decay
	cfg.FadeDuration = 0.01               // 10ms fade in/out
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}

// NewLinnDrum creates a LinnDrum-style kick drum sound
func NewLinnDrum(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(60.0, 40.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.01 // Smooth attack
	cfg.Decay = 0.5   // Moderate decay for punchiness
	cfg.Sustain = 0.1 // Low sustain for a tight sound
	cfg.Release = 0.3
	cfg.Drive = 0.4
	cfg.FilterCutoff = 5000 // Balanced cutoff for clarity
	cfg.Sweep = 0.6
	cfg.PitchDecay = 0.4                  // Gentle pitch decay for depth
	cfg.FadeDuration = 0.02               // 20ms fade in/out for a punchy yet smooth sound
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}

// NewDeepHouse creates a Deep House kick drum
func NewDeepHouse(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(45.0, 25.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.005                    // Soft attack for a smooth start
	cfg.Decay = 0.9                       // Long decay for a deep, lingering sound
	cfg.Sustain = 0.3                     // Medium sustain for warmth
	cfg.Release = 0.7                     // Extended release for a smooth tail
	cfg.Drive = 0.6                       // Moderate drive for warmth and body
	cfg.FilterCutoff = 3500               // Low cutoff for deep, smooth bass
	cfg.Sweep = 0.8                       // Gentle sweep to keep it subtle
	cfg.PitchDecay = 0.6                  // Slight pitch decay for that deep house feel
	cfg.FadeDuration = 0.03               // 30ms fade in/out for a more gradual sound
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}

// New606 creates a 606-style kick drum
func New606(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(65.0, 45.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.01
	cfg.Decay = 0.3
	cfg.Sustain = 0.1
	cfg.Release = 0.2
	cfg.Drive = 0.4
	cfg.FilterCutoff = 5000
	cfg.Sweep = 0.7
	cfg.PitchDecay = 0.5
	cfg.FadeDuration = 0.015              // 15ms fade in/out for a balanced sound
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}

// New707 creates a 707-style kick drum
func New707(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(60.0, 40.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveTriangle
	cfg.Attack = 0.005
	cfg.Decay = 0.3
	cfg.Sustain = 0.2
	cfg.Release = 0.2
	cfg.Drive = 0.3
	cfg.FilterCutoff = 5000
	cfg.Sweep = 0.6
	cfg.PitchDecay = 0.3
	cfg.FadeDuration = 0.01               // 10ms fade in/out
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}

// New808 creates an 808-style kick drum
func New808(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(55.0, 30.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.01
	cfg.Decay = 0.8
	cfg.Sustain = 0.2
	cfg.Release = 0.6
	cfg.Drive = 0.2
	cfg.FilterCutoff = 4000
	cfg.Sweep = 0.9
	cfg.PitchDecay = 0.5
	cfg.FadeDuration = 0.02               // 20ms fade in/out for a fuller sound
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}

// New909 creates a 909-style kick drum
func New909(sampleRate int, duration float64, bitDepth int, output io.WriteSeeker) (*Settings, error) {
	cfg, err := NewSettings(70.0, 50.0, sampleRate, duration, bitDepth, output)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveTriangle
	cfg.Attack = 0.002
	cfg.Decay = 0.2
	cfg.Sustain = 0.1
	cfg.Release = 0.3
	cfg.Drive = 0.4
	cfg.FilterCutoff = 8000
	cfg.Sweep = 0.7
	cfg.PitchDecay = 0.2
	cfg.FadeDuration = 0.015              // 15ms fade in/out to balance between smoothness and clarity
	cfg.SmoothFrequencyTransitions = true // Enable smooth frequency transitions

	return cfg, nil
}
