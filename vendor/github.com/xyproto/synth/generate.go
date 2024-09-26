package synth

import (
	"fmt"
	"math"
	"math/rand"
)

// GenerateClap generates a clap sound by combining filtered noise bursts
func (cfg *Settings) GenerateClap() ([]float64, error) {
	// Clap consists of multiple bursts of filtered noise
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Clap is typically 3 distinct noise bursts with delays between them
	burstCount := 3
	delayBetweenBursts := 0.02 // 20ms between bursts
	burstDuration := (cfg.Duration - (float64(burstCount-1) * delayBetweenBursts)) / float64(burstCount)

	for burst := 0; burst < burstCount; burst++ {
		startIndex := int(float64(burst) * delayBetweenBursts * float64(cfg.SampleRate))
		burstSamples := int(burstDuration * float64(cfg.SampleRate))

		// Generate filtered white noise for each burst
		burstNoise := GenerateWhiteNoise(burstSamples, cfg.NoiseAmount)
		burstNoise = LowPassFilter(burstNoise, cfg.FilterCutoff, cfg.SampleRate)

		// Apply ADSR envelope to each burst
		burstNoise = ApplyEnvelope(burstNoise, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

		// Mix the bursts into the final sample array
		for i := 0; i < burstSamples; i++ {
			if startIndex+i < numSamples {
				samples[startIndex+i] += burstNoise[i]
			}
		}
	}

	// Apply limiter to ensure the final clap sound is within [-1, 1]
	samples = Limiter(samples)

	return samples, nil
}

// GenerateSnare generates a snare drum sound by combining noise and a tonal component
func (cfg *Settings) GenerateSnare() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate the tonal part (body of the snare) using a short burst of a tuned waveform
	tonalSamples := int(cfg.Duration * float64(cfg.SampleRate) * 0.3) // 30% of the duration is tonal
	frequencyDecayFactor := 0.99                                      // Decay factor for pitch modulation

	for i := 0; i < tonalSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration) * frequencyDecayFactor
		// Use a waveform type like Sawtooth or Square to generate the tonal body
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
		default:
			return nil, fmt.Errorf("unsupported waveform type: %d", cfg.WaveformType)
		}
		samples[i] += sample
	}

	// Generate the noise part (snare "rattle") using filtered pink noise
	noiseSamples := GeneratePinkNoise(numSamples, cfg.NoiseAmount)
	noiseSamples = BandPassFilter(noiseSamples, 150.0, 8000.0, cfg.SampleRate) // Bandpass to shape the noise

	// Mix noise with the tonal part
	for i := 0; i < numSamples; i++ {
		samples[i] += noiseSamples[i]
	}

	// Apply ADSR envelope to shape the sound
	samples = ApplyEnvelope(samples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Apply drive (distortion) to add more punch to the snare
	samples = Drive(samples, cfg.Drive)

	// Apply limiter to keep everything within the [-1, 1] range
	samples = Limiter(samples)

	return samples, nil
}

// GenerateClosedHH generates a closed hi-hat sound using filtered noise
func (cfg *Settings) GenerateClosedHH() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate the noise component (hi-hat is mostly metallic noise)
	noiseSamples := GenerateWhiteNoise(numSamples, cfg.NoiseAmount)

	// Apply a high-pass filter to emphasize the high frequencies of the hi-hat sound
	noiseSamples = HighPassFilter(noiseSamples, 5000.0, cfg.SampleRate) // Remove low frequencies below 5kHz

	// Optionally, apply a band-pass filter to focus the hi-hat frequency range
	noiseSamples = BandPassFilter(noiseSamples, 5000.0, 10000.0, cfg.SampleRate) // Focus on higher frequencies

	// Mix the filtered noise into the final samples
	for i := 0; i < numSamples; i++ {
		samples[i] += noiseSamples[i]
	}

	// Apply a very short ADSR envelope to create the sharp, percussive nature of a closed hi-hat
	samples = ApplyEnvelope(samples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Add some drive (distortion) to give the hi-hat a metallic, sharp edge
	samples = Drive(samples, cfg.Drive)

	// Apply a fade-out to give the hi-hat a quick decay, typical of a closed hi-hat
	samples = ApplyFadeOut(samples, cfg.FadeDuration, cfg.SampleRate)

	// Limit the amplitude to avoid clipping
	samples = Limiter(samples)

	return samples, nil
}

// GenerateOpenHH generates an open hi-hat sound using filtered noise
func (cfg *Settings) GenerateOpenHH() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate the noise component (hi-hat is mostly metallic noise)
	noiseSamples := GenerateWhiteNoise(numSamples, cfg.NoiseAmount)

	// Apply a high-pass filter to emphasize the high frequencies of the hi-hat sound
	noiseSamples = HighPassFilter(noiseSamples, 5000.0, cfg.SampleRate) // Remove low frequencies below 5kHz

	// Optionally, apply a band-pass filter to focus the hi-hat frequency range
	noiseSamples = BandPassFilter(noiseSamples, 5000.0, 10000.0, cfg.SampleRate) // Focus on higher frequencies

	// Mix the filtered noise into the final samples
	for i := 0; i < numSamples; i++ {
		samples[i] += noiseSamples[i]
	}

	// Apply a longer ADSR envelope to create the open, sustained nature of the open hi-hat
	samples = ApplyEnvelope(samples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Add some drive (distortion) to give the hi-hat a metallic, sharp edge
	samples = Drive(samples, cfg.Drive)

	// Apply a fade-out to simulate the natural decay of an open hi-hat
	samples = ApplyFadeOut(samples, cfg.FadeDuration, cfg.SampleRate)

	// Limit the amplitude to avoid clipping
	samples = Limiter(samples)

	return samples, nil
}

// GenerateRimshot generates a rimshot sound by using a short burst of high-frequency noise
func (cfg *Settings) GenerateRimshot() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)

	// Generate a sharp, metallic noise burst for the rimshot
	noiseSamples := GenerateWhiteNoise(numSamples, cfg.NoiseAmount)

	// Apply a band-pass filter to focus the rimshot on high-mid frequencies
	noiseSamples = BandPassFilter(noiseSamples, 2000.0, 6000.0, cfg.SampleRate)

	// Apply a short ADSR envelope to make it a quick, percussive sound
	samples := ApplyEnvelope(noiseSamples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Add drive (distortion) for punch
	samples = Drive(samples, cfg.Drive)

	// Apply limiter to prevent clipping
	samples = Limiter(samples)

	return samples, nil
}

// GenerateTom generates a tom drum sound, configurable for low, mid, and high toms
func (cfg *Settings) GenerateTom() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate a decaying sine wave to represent the tom's body
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration)
		sample := math.Sin(2 * math.Pi * frequency * t)
		samples[i] = sample
	}

	// Apply ADSR envelope to shape the sound of the tom
	samples = ApplyEnvelope(samples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Add a bit of pink noise to simulate drum head vibrations
	noiseSamples := GeneratePinkNoise(numSamples, cfg.NoiseAmount)
	noiseSamples = LowPassFilter(noiseSamples, cfg.FilterCutoff, cfg.SampleRate)
	for i := 0; i < numSamples; i++ {
		samples[i] += noiseSamples[i] * 0.2 // Slight noise mixed in
	}

	// Apply drive (distortion) to give the tom more depth
	samples = Drive(samples, cfg.Drive)

	// Apply limiter to keep the sound within [-1, 1] range
	samples = Limiter(samples)

	return samples, nil
}

// GeneratePercussion generates a tonal percussion sound like bongo or conga
func (cfg *Settings) GeneratePercussion() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate the tonal part using a high-pitched sine or triangle wave
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration)
		sample := math.Sin(2 * math.Pi * frequency * t)
		samples[i] = sample
	}

	// Apply a quick, snappy ADSR envelope for the short percussion hit
	samples = ApplyEnvelope(samples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Add a small amount of pink noise for texture
	noiseSamples := GeneratePinkNoise(numSamples, cfg.NoiseAmount)
	noiseSamples = BandPassFilter(noiseSamples, 300.0, 1000.0, cfg.SampleRate)

	for i := 0; i < numSamples; i++ {
		samples[i] += noiseSamples[i] * 0.1 // Lightly mix in noise
	}

	// Apply drive to enhance the percussive attack
	samples = Drive(samples, cfg.Drive)

	// Apply limiter to keep the sound within [-1, 1] range
	samples = Limiter(samples)

	return samples, nil
}

// GenerateRide generates a ride cymbal sound using filtered noise
func (cfg *Settings) GenerateRide() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)

	// Generate metallic noise for the ride cymbal
	noiseSamples := GenerateWhiteNoise(numSamples, cfg.NoiseAmount)

	// Apply a high-pass filter to keep the ride focused on high frequencies
	noiseSamples = HighPassFilter(noiseSamples, 5000.0, cfg.SampleRate)

	// Apply a longer ADSR envelope to simulate the ringing sound of a ride cymbal
	samples := ApplyEnvelope(noiseSamples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Apply drive (distortion) for added metallic resonance
	samples = Drive(samples, cfg.Drive)

	// Apply limiter to ensure the output stays in the [-1, 1] range
	samples = Limiter(samples)

	return samples, nil
}

// GenerateCrash generates a crash cymbal sound using filtered noise
func (cfg *Settings) GenerateCrash() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)

	// Generate wide-spectrum noise for the crash cymbal
	noiseSamples := GenerateWhiteNoise(numSamples, cfg.NoiseAmount)

	// Apply a band-pass filter to focus on the metallic frequency range
	noiseSamples = BandPassFilter(noiseSamples, 2000.0, 15000.0, cfg.SampleRate)

	// Apply a quick attack and a longer decay ADSR envelope
	samples := ApplyEnvelope(noiseSamples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Add drive to enhance the "explosive" nature of the crash
	samples = Drive(samples, cfg.Drive)

	// Apply limiter to keep the sound within the [-1, 1] range
	samples = Limiter(samples)

	return samples, nil
}

// GenerateKick generates the kick waveform and returns it as a slice of float64 samples (without writing to disk).
func (cfg *Settings) GenerateKick() ([]float64, error) {
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
			sample = GenerateWhiteNoise(1, cfg.NoiseAmount)[0]
		case WavePinkNoise:
			sample = GeneratePinkNoise(1, cfg.NoiseAmount)[0]
		case WaveBrownNoise:
			sample = GenerateBrownNoise(1, cfg.NoiseAmount)[0]
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
			sample = GenerateWhiteNoise(1, cfg.NoiseAmount)[0]
		case WavePinkNoise:
			sample = GeneratePinkNoise(1, cfg.NoiseAmount)[0]
		case WaveBrownNoise:
			sample = GenerateBrownNoise(1, cfg.NoiseAmount)[0]
		default:
			return nil, fmt.Errorf("unsupported waveform type: %d", cfg.WaveformType)
		}

		samples[i] = sample
	}

	return samples, nil
}

// Generate is a wrapper function that calls the appropriate Generate* function based on the input string t
func (cfg *Settings) Generate(t string) ([]float64, error) {
	switch t {
	case "kick":
		return cfg.GenerateKick()
	case "clap":
		return cfg.GenerateClap()
	case "snare":
		return cfg.GenerateSnare()
	case "closed_hh":
		return cfg.GenerateClosedHH()
	case "open_hh":
		return cfg.GenerateOpenHH()
	case "rimshot":
		return cfg.GenerateRimshot()
	case "tom":
		return cfg.GenerateTom()
	case "percussion":
		return cfg.GeneratePercussion()
	case "ride":
		return cfg.GenerateRide()
	case "crash":
		return cfg.GenerateCrash()
	case "bass":
		return cfg.GenerateBass()
	case "xylophone":
		return cfg.GenerateXylophone()
	case "lead":
		return cfg.GenerateLead()
	default:
		return nil, fmt.Errorf("unknown sound type: %s", t)
	}
}

// GenerateWhiteNoise generates white noise
func GenerateWhiteNoise(length int, amount float64) []float64 {
	noise := make([]float64, length)
	for i := range noise {
		noise[i] = (rand.Float64()*2 - 1) * amount
	}
	return noise
}

// GeneratePinkNoise generates pink noise
func GeneratePinkNoise(length int, amount float64) []float64 {
	noise := make([]float64, length)
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
	return noise
}

// GenerateBrownNoise generates brown noise
func GenerateBrownNoise(length int, amount float64) []float64 {
	noise := make([]float64, length)
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
	return noise
}

// GenerateBass generates a deep, detuned bass sound typical of deep house
func (cfg *Settings) GenerateBass() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)

	// Generate detuned sawtooth oscillators for a deep bass sound
	detune := []float64{-0.01, 0.01} // Slight detuning for a thicker sound
	bassWave := DetunedOscillators(cfg.StartFreq, detune, numSamples, cfg.SampleRate)

	// Apply a low-pass filter to keep the bass deep and focused on lower frequencies
	bassWave = LowPassFilter(bassWave, 150.0, cfg.SampleRate) // Low-pass at 150Hz for deep bass

	// Apply ADSR envelope for bass dynamics
	bassWave = ApplyEnvelope(bassWave, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Apply drive to give the bass some extra punch and warmth
	bassWave = Drive(bassWave, cfg.Drive)

	// Limit the amplitude to avoid clipping
	bassWave = Limiter(bassWave)

	return bassWave, nil
}

// GenerateXylophone generates a xylophone-like sound for arpeggios
func (cfg *Settings) GenerateXylophone() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)
	samples := make([]float64, numSamples)

	// Generate a sine wave to simulate the xylophone's tonal character
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(cfg.SampleRate)
		frequency := cfg.StartFreq * math.Pow(cfg.EndFreq/cfg.StartFreq, t/cfg.Duration)
		sample := math.Sin(2 * math.Pi * frequency * t)
		samples[i] = sample
	}

	// Apply a short, sharp ADSR envelope to simulate the percussive attack of a xylophone
	samples = ApplyEnvelope(samples, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Optionally, apply a bit of reverb for depth
	samples, _ = SchroederReverb(samples, 0.3, []int{1557, 1617, 1491, 1422}, []int{225, 556})

	// Apply limiter to keep everything within the [-1, 1] range
	samples = Limiter(samples)

	return samples, nil
}

// GenerateLead generates a bright, detuned lead sound
func (cfg *Settings) GenerateLead() ([]float64, error) {
	numSamples := int(float64(cfg.SampleRate) * cfg.Duration)

	// Generate detuned sawtooth oscillators for a bright lead sound
	detune := []float64{-0.02, 0.02} // Slight detuning for a rich, thick sound
	leadWave := DetunedOscillators(cfg.StartFreq, detune, numSamples, cfg.SampleRate)

	// Apply an ADSR envelope for the lead sound dynamics
	leadWave = ApplyEnvelope(leadWave, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.SampleRate)

	// Optionally, apply frequency modulation for a more expressive lead
	leadWave = ApplyFrequencyModulation(leadWave, 5.0, 0.05, cfg.SampleRate) // Slow modulation

	// Apply drive for extra brightness and character
	leadWave = Drive(leadWave, cfg.Drive)

	// Limit the amplitude to avoid clipping
	leadWave = Limiter(leadWave)

	return leadWave, nil
}
