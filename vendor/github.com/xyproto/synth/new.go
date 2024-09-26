package synth

import (
	"errors"
	"io"
	"math/rand"
)

// NewSettings creates a new Settings instance with default values for a percussive sound
func NewSettings(output io.WriteSeeker, startFreq, endFreq, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	if sampleRate <= 0 || duration <= 0 || channels <= 0 {
		return nil, errors.New("invalid sample rate, duration or channels")
	}
	return &Settings{
		SampleRate:                 sampleRate,
		BitDepth:                   bitDepth, // Audio bit depth (8, 16, 24, or 32 bits)
		Channels:                   channels,
		Output:                     output,                           // Output writer
		StartFreq:                  startFreq,                        // Starting frequency (Hz)
		EndFreq:                    endFreq,                          // Ending frequency (Hz)
		Duration:                   duration,                         // Duration in seconds
		WaveformType:               WaveSine,                         // Default waveform type
		NoiseAmount:                0.0,                              // Default noise amount
		Attack:                     0.005,                            // Attack time in seconds
		Decay:                      0.2,                              // Decay time in seconds
		Sustain:                    0.1,                              // Sustain level
		Release:                    0.1,                              // Release time in seconds
		Drive:                      0.2,                              // Drive (distortion) amount
		FilterCutoff:               8000,                             // Filter cutoff frequency (Hz)
		Sweep:                      0.7,                              // Pitch sweep amount
		PitchDecay:                 0.4,                              // Pitch envelope decay time
		NumOscillators:             1,                                // Default to 1 oscillator
		OscillatorLevels:           []float64{1.0},                   // Default oscillator level
		SaturatorAmount:            0.3,                              // Saturation amount
		FilterResonance:            1.0,                              // Filter resonance
		FilterBands:                []float64{200.0, 1000.0, 3000.0}, // Multi-band filter frequencies
		FadeDuration:               0.01,                             // Fade in/out duration in seconds
		SmoothFrequencyTransitions: true,                             // Enable smooth frequency transitions by default
	}, nil
}

// NewRandomKick generates random settings for experimental kick drum sounds
func NewRandomKick(output io.WriteSeeker, sampleRate, bitDepth, channels int) *Settings {
	cfg, _ := NewSettings(output, 50.0, 30.0, 1.0, sampleRate, bitDepth, channels)
	cfg.Attack = rand.Float64() * 0.02
	cfg.Decay = 0.1 + rand.Float64()*0.5
	cfg.Sustain = rand.Float64() * 0.5
	cfg.Release = 0.1 + rand.Float64()*0.5
	cfg.Drive = rand.Float64()
	cfg.FilterCutoff = 2000 + rand.Float64()*6000
	cfg.FilterResonance = 1.0 + rand.Float64()*1.0
	cfg.Sweep = rand.Float64() * 1.5
	cfg.PitchDecay = rand.Float64() * 1.5
	cfg.FadeDuration = rand.Float64() * 0.1
	cfg.SmoothFrequencyTransitions = rand.Float64() >= 0.1
	if rand.Float64() < 0.1 {
		cfg.WaveformType = rand.Intn(7)
	} else {
		cfg.WaveformType = rand.Intn(2)
	}
	return cfg
}

// NewSnareSettings creates a new Settings instance with predefined values optimized for snare drum sounds.
func NewSnareSettings(output io.WriteSeeker, sampleRate, bitDepth, channels int) (*Settings, error) {
	if sampleRate <= 0 || bitDepth <= 0 || channels <= 0 {
		return nil, errors.New("invalid sample rate, bit depth, or channels")
	}

	return &Settings{
		SampleRate:                 sampleRate,
		BitDepth:                   bitDepth,   // e.g., 16
		Channels:                   channels,   // e.g., 1 for mono, 2 for stereo
		Output:                     output,     // Output writer
		StartFreq:                  300.0,      // Starting frequency in Hz
		EndFreq:                    100.0,      // Ending frequency in Hz (rapid pitch drop)
		Duration:                   0.5,        // Short duration for snare snap
		WaveformType:               WaveSquare, // Square waveform for a brighter sound
		NoiseAmount:                0.7,        // Significant noise for snare rattle
		Attack:                     0.005,      // Very quick attack
		Decay:                      0.1,        // Short decay
		Sustain:                    0.0,        // No sustain
		Release:                    0.05,       // Short release
		Drive:                      0.3,        // Moderate drive for slight distortion
		FilterCutoff:               5000.0,     // High cutoff to retain brightness
		Sweep:                      0.5,        // Moderate pitch sweep
		PitchDecay:                 0.3,        // Pitch envelope decay
		NumOscillators:             1,          // Single oscillator
		OscillatorLevels:           []float64{1.0},
		SaturatorAmount:            0.2,                              // Mild saturation
		FilterResonance:            1.0,                              // Standard resonance
		FilterBands:                []float64{500.0, 2000.0, 6000.0}, // Example multi-band frequencies
		FadeDuration:               0.01,                             // Quick fade to prevent clicks
		SmoothFrequencyTransitions: true,                             // Enable smooth transitions
	}, nil
}

// NewRandomSnare generates random settings for experimental snare drum sounds
func NewRandomSnare(output io.WriteSeeker, sampleRate, bitDepth, channels int) *Settings {
	cfg, _ := NewSettings(output, 300.0, 100.0, 0.5, sampleRate, bitDepth, channels)
	cfg.NoiseAmount = 0.5 + rand.Float64()*0.5
	cfg.Attack = rand.Float64() * 0.01
	cfg.Decay = 0.05 + rand.Float64()*0.2
	cfg.Sustain = 0.0
	cfg.Release = 0.05 + rand.Float64()*0.2
	cfg.Drive = rand.Float64()
	cfg.FilterCutoff = 5000 + rand.Float64()*3000
	cfg.FilterResonance = 1.0 + rand.Float64()*1.0
	cfg.FadeDuration = rand.Float64() * 0.05
	cfg.SmoothFrequencyTransitions = rand.Float64() >= 0.1
	cfg.WaveformType = rand.Intn(4)
	return cfg
}

// New606Kick creates a 606-style kick drum
func New606Kick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 80.0, 50.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.002
	cfg.Decay = 0.25
	cfg.Sustain = 0.0
	cfg.Release = 0.15
	cfg.Drive = 0.3
	cfg.FilterCutoff = 4000
	cfg.FilterResonance = 1.2
	cfg.Sweep = 0.8
	cfg.PitchDecay = 0.6
	cfg.FadeDuration = 0.008
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// New707Kick creates a 707-style kick drum
func New707Kick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 85.0, 55.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveTriangle
	cfg.Attack = 0.003
	cfg.Decay = 0.3
	cfg.Sustain = 0.0
	cfg.Release = 0.2
	cfg.Drive = 0.4
	cfg.FilterCutoff = 4500
	cfg.FilterResonance = 1.0
	cfg.Sweep = 0.7
	cfg.PitchDecay = 0.5
	cfg.FadeDuration = 0.01
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// New808Kick creates an 808-style kick drum with improved parameters
func New808Kick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 55.0, 30.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.0 // Instant attack for a punchier sound
	cfg.Decay = 0.6
	cfg.Sustain = 0.0
	cfg.Release = 0.4
	cfg.Drive = 0.5
	cfg.FilterCutoff = 2000 // Lower cutoff for deeper bass
	cfg.FilterResonance = 1.5
	cfg.Sweep = 1.0
	cfg.PitchDecay = 0.8
	cfg.FadeDuration = 0.005
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// New909Kick creates a 909-style kick drum with enhanced punch
func New909Kick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 60.0, 50.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveTriangle
	cfg.Attack = 0.0
	cfg.Decay = 0.25
	cfg.Sustain = 0.0
	cfg.Release = 0.2
	cfg.Drive = 0.7
	cfg.FilterCutoff = 2500
	cfg.FilterResonance = 1.4
	cfg.Sweep = 0.8
	cfg.PitchDecay = 0.6
	cfg.FadeDuration = 0.005
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// NewLinnDrumKick creates a LinnDrum-style kick drum sound
func NewLinnDrumKick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 75.0, 45.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSquare
	cfg.Attack = 0.004
	cfg.Decay = 0.35
	cfg.Sustain = 0.0
	cfg.Release = 0.25
	cfg.Drive = 0.5
	cfg.FilterCutoff = 3000
	cfg.FilterResonance = 1.0
	cfg.Sweep = 0.6
	cfg.PitchDecay = 0.5
	cfg.FadeDuration = 0.015
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// NewDeepHouseKick creates a Deep House kick drum with richer bass
func NewDeepHouseKick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 50.0, 30.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.Attack = 0.005
	cfg.Decay = 1.0
	cfg.Sustain = 0.0
	cfg.Release = 0.8
	cfg.Drive = 0.6
	cfg.FilterCutoff = 1500
	cfg.FilterResonance = 1.2
	cfg.Sweep = 0.9
	cfg.PitchDecay = 0.7
	cfg.FadeDuration = 0.02
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// NewExperimentalKick creates an experimental kick drum sound
func NewExperimentalKick(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 70.0, 20.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSawtooth
	cfg.Attack = 0.001
	cfg.Decay = 0.5
	cfg.Sustain = 0.0
	cfg.Release = 0.3
	cfg.Drive = 0.9
	cfg.FilterCutoff = 5000
	cfg.FilterResonance = 1.5
	cfg.Sweep = 1.2
	cfg.PitchDecay = 0.9
	cfg.FadeDuration = 0.01
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// Snare Drum Functions

// New606Snare creates a snare drum sound similar to the Roland TR-606
func New606Snare(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 350.0, 200.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveTriangle
	cfg.NoiseAmount = 0.6
	cfg.Attack = 0.005
	cfg.Decay = 0.15
	cfg.Sustain = 0.0
	cfg.Release = 0.1
	cfg.Drive = 0.3
	cfg.FilterCutoff = 7000
	cfg.FilterResonance = 1.2
	cfg.FadeDuration = 0.01
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// New707Snare creates a snare drum sound similar to the Roland TR-707
func New707Snare(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 300.0, 150.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSquare
	cfg.NoiseAmount = 0.5
	cfg.Attack = 0.001
	cfg.Decay = 0.2
	cfg.Sustain = 0.0
	cfg.Release = 0.15
	cfg.Drive = 0.4
	cfg.FilterCutoff = 6500
	cfg.FilterResonance = 1.0
	cfg.FadeDuration = 0.008
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// New808Snare creates a snare drum sound similar to the Roland TR-808
func New808Snare(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 240.0, 120.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSine
	cfg.NoiseAmount = 0.7
	cfg.Attack = 0.002
	cfg.Decay = 0.25
	cfg.Sustain = 0.0
	cfg.Release = 0.2
	cfg.Drive = 0.5
	cfg.FilterCutoff = 6000
	cfg.FilterResonance = 1.5
	cfg.FadeDuration = 0.015
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// New909Snare creates a snare drum sound similar to the Roland TR-909
func New909Snare(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 250.0, 130.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSawtooth
	cfg.NoiseAmount = 0.6
	cfg.Attack = 0.003
	cfg.Decay = 0.18
	cfg.Sustain = 0.0
	cfg.Release = 0.15
	cfg.Drive = 0.6
	cfg.FilterCutoff = 7000
	cfg.FilterResonance = 1.3
	cfg.FadeDuration = 0.012
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// NewLinnDrumSnare creates a snare drum sound similar to the LinnDrum
func NewLinnDrumSnare(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 260.0, 140.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSquare
	cfg.NoiseAmount = 0.5
	cfg.Attack = 0.001
	cfg.Decay = 0.22
	cfg.Sustain = 0.0
	cfg.Release = 0.18
	cfg.Drive = 0.4
	cfg.FilterCutoff = 7500
	cfg.FilterResonance = 1.1
	cfg.FadeDuration = 0.01
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}

// NewExperimentalSnare creates an experimental snare drum sound
func NewExperimentalSnare(output io.WriteSeeker, duration float64, sampleRate, bitDepth, channels int) (*Settings, error) {
	cfg, err := NewSettings(output, 400.0, 100.0, duration, sampleRate, bitDepth, channels)
	if err != nil {
		return nil, err
	}
	cfg.WaveformType = WaveSawtooth
	cfg.NoiseAmount = 0.8
	cfg.Attack = 0.002
	cfg.Decay = 0.3
	cfg.Sustain = 0.0
	cfg.Release = 0.25
	cfg.Drive = 0.7
	cfg.FilterCutoff = 5000
	cfg.FilterResonance = 1.7
	cfg.FadeDuration = 0.02
	cfg.SmoothFrequencyTransitions = true
	return cfg, nil
}
