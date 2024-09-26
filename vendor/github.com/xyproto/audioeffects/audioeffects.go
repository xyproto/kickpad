package audioeffects

import (
	"math"
	"math/rand"
)

func lowPassCoefficients(filterType string, freq, Q, sampleRate float64) (float64, float64, float64, float64, float64) {
	omega := 2 * math.Pi * freq / sampleRate
	sinOmega := math.Sin(omega)
	cosOmega := math.Cos(omega)
	alpha := sinOmega / (2 * Q)
	var a0, a1, a2, b1, b2 float64
	switch filterType {
	case "low-pass":
		a0 = 1 + alpha
		a1 = -2 * cosOmega
		a2 = 1 - alpha
		b1 = 2 * cosOmega
		b2 = 1 - alpha
	case "high-pass":
		a0 = 1 + alpha
		a1 = -2 * cosOmega
		a2 = 1 - alpha
		b1 = 2 * cosOmega
		b2 = 1 - alpha
	case "band-pass":
		a0 = 1 + alpha
		a1 = -2 * cosOmega
		a2 = 1 - alpha
		b1 = 2 * cosOmega
		b2 = 1 - alpha
	case "notch":
		a0 = 1 + alpha
		a1 = -2 * cosOmega
		a2 = 1 - alpha
		b1 = 2 * cosOmega
		b2 = 1 - alpha
	case "all-pass":
		a0 = 1 + alpha
		a1 = -2 * cosOmega
		a2 = 1 - alpha
		b1 = 2 * cosOmega
		b2 = -(1 - alpha)
	default:
		a0 = 1
		a1 = 0
		a2 = 0
		b1 = 0
		b2 = 0
	}
	a0 /= a0
	a1 /= a0
	a2 /= a0
	b1 /= a0
	b2 /= a0
	return a0, a1, a2, b1, b2
}

func BiquadFilter(samples []float64, filterType string, freq, Q, sampleRate float64) []float64 {
	a0, a1, a2, b1, b2 := lowPassCoefficients(filterType, freq, Q, sampleRate)
	x1, x2, y1, y2 := 0.0, 0.0, 0.0, 0.0
	filtered := make([]float64, len(samples))
	for i, sample := range samples {
		y := a0*sample + a1*x1 + a2*x2 - b1*y1 - b2*y2
		filtered[i] = y
		x2 = x1
		x1 = sample
		y2 = y1
		y1 = y
	}
	return filtered
}

func FadeIn(samples []float64, duration float64, sampleRate int) []float64 {
	faded := make([]float64, len(samples))
	fadeSamples := int(duration * float64(sampleRate))
	if fadeSamples > len(samples) {
		fadeSamples = len(samples)
	}
	for i := 0; i < len(samples); i++ {
		if i < fadeSamples {
			faded[i] = samples[i] * float64(i) / float64(fadeSamples)
		} else {
			faded[i] = samples[i]
		}
	}
	return faded
}

func FadeOut(samples []float64, duration float64, sampleRate int) []float64 {
	faded := make([]float64, len(samples))
	fadeSamples := int(duration * float64(sampleRate))
	startFade := len(samples) - fadeSamples
	if startFade < 0 {
		startFade = 0
	}
	for i := 0; i < len(samples); i++ {
		if i >= startFade {
			faded[i] = samples[i] * float64(len(samples)-i) / float64(fadeSamples)
		} else {
			faded[i] = samples[i]
		}
	}
	return faded
}

func LowPassFilter(samples []float64, cutoffFreq float64, sampleRate int) []float64 {
	return BiquadFilter(samples, "low-pass", cutoffFreq, 0.707, float64(sampleRate))
}

func HighPassFilter(samples []float64, cutoffFreq float64, sampleRate int) []float64 {
	return BiquadFilter(samples, "high-pass", cutoffFreq, 0.707, float64(sampleRate))
}

func BandPassFilter(samples []float64, lowFreq, highFreq float64, sampleRate int) []float64 {
	centerFreq := (lowFreq + highFreq) / 2
	BandWidth := highFreq - lowFreq
	Q := centerFreq / BandWidth
	return BiquadFilter(samples, "band-pass", centerFreq, Q, float64(sampleRate))
}

func NoiseGate(samples []float64, threshold, attack, release float64, sampleRate int) []float64 {
	gated := make([]float64, len(samples))
	gain := 1.0
	attackCoeff := math.Exp(-1.0 / (attack * float64(sampleRate)))
	releaseCoeff := math.Exp(-1.0 / (release * float64(sampleRate)))
	for i, sample := range samples {
		absSample := math.Abs(sample)
		if absSample > threshold {
			gain = attackCoeff*gain + (1.0-attackCoeff)*1.0
		} else {
			gain = releaseCoeff*gain + (1.0-releaseCoeff)*0.0
		}
		gated[i] = sample * gain
	}
	return gated
}

func StereoDelay(left, right []float64, sampleRate int, delayTimeLeft, delayTimeRight float64, feedback, mix float64) ([]float64, []float64) {
	delaySamplesLeft := int(delayTimeLeft * float64(sampleRate))
	delaySamplesRight := int(delayTimeRight * float64(sampleRate))
	if delaySamplesLeft <= 0 {
		delaySamplesLeft = 1
	}
	if delaySamplesRight <= 0 {
		delaySamplesRight = 1
	}
	delayedLeft := make([]float64, len(left))
	delayedRight := make([]float64, len(right))
	bufferLeft := make([]float64, delaySamplesLeft)
	bufferRight := make([]float64, delaySamplesRight)
	bufferIndexLeft, bufferIndexRight := 0, 0
	for i := 0; i < len(left); i++ {
		delayedSampleLeft := bufferLeft[bufferIndexLeft]
		delayedLeft[i] = left[i]*(1-mix) + delayedSampleLeft*mix
		bufferLeft[bufferIndexLeft] = left[i] + delayedSampleLeft*feedback
		bufferIndexLeft = (bufferIndexLeft + 1) % delaySamplesLeft
		delayedSampleRight := bufferRight[bufferIndexRight]
		delayedRight[i] = right[i]*(1-mix) + delayedSampleRight*mix
		bufferRight[bufferIndexRight] = right[i] + delayedSampleRight*feedback
		bufferIndexRight = (bufferIndexRight + 1) % delaySamplesRight
	}
	return delayedLeft, delayedRight
}

func Expander(samples []float64, threshold, ratio, attack, release float64, sampleRate int) []float64 {
	expanded := make([]float64, len(samples))
	envelope := 0.0
	attackCoeff := math.Exp(-1.0 / (attack * float64(sampleRate)))
	releaseCoeff := math.Exp(-1.0 / (release * float64(sampleRate)))
	for i, sample := range samples {
		absSample := math.Abs(sample)
		if absSample > envelope {
			envelope = attackCoeff*envelope + (1.0-attackCoeff)*absSample
		} else {
			envelope = releaseCoeff*envelope + (1.0-releaseCoeff)*absSample
		}
		var gain float64
		if envelope < threshold {
			gain = envelope * ratio / threshold
			if gain > 1.0 {
				gain = 1.0
			}
		} else {
			gain = 1.0
		}
		expanded[i] = sample * gain
	}
	return expanded
}

func SoftClip(sample, drive float64) float64 {
	return (3 + drive) * sample / (1 + drive*math.Abs(sample))
}

func SoftClippingDistortion(samples []float64, drive float64) []float64 {
	distorted := make([]float64, len(samples))
	for i, sample := range samples {
		distorted[i] = SoftClip(sample, drive)
		if distorted[i] > 1.0 {
			distorted[i] = 1.0
		} else if distorted[i] < -1.0 {
			distorted[i] = -1.0
		}
	}
	return distorted
}

func SidechainCompressor(target, trigger []float64, threshold, ratio, attack, release float64, sampleRate int) []float64 {
	if len(target) != len(trigger) {
		return target
	}
	compressed := make([]float64, len(target))
	gain := 1.0
	attackCoeff := math.Exp(-1.0 / (attack * float64(sampleRate)))
	releaseCoeff := math.Exp(-1.0 / (release * float64(sampleRate)))
	for i := 0; i < len(target); i++ {
		absTrigger := math.Abs(trigger[i])
		if absTrigger > gain {
			gain = attackCoeff*gain + (1.0-attackCoeff)*absTrigger
		} else {
			gain = releaseCoeff*gain + (1.0-releaseCoeff)*absTrigger
		}
		if gain > threshold {
			gain = threshold + (gain-threshold)/ratio
			gain /= gain
			if gain < 0.0 {
				gain = 0.0
			}
		} else {
			gain = 1.0
		}
		compressed[i] = target[i] * gain
	}
	return compressed
}

func Compressor(samples []float64, threshold, ratio, attack, release float64, sampleRate int) []float64 {
	compressed := make([]float64, len(samples))
	gain := 1.0
	attackCoeff := math.Exp(-1.0 / (attack * float64(sampleRate)))
	releaseCoeff := math.Exp(-1.0 / (release * float64(sampleRate)))
	for i, sample := range samples {
		absSample := math.Abs(sample)
		if absSample > threshold {
			gain = attackCoeff*gain + (1.0-attackCoeff)*(threshold/absSample)
		} else {
			gain = releaseCoeff*gain + (1.0-releaseCoeff)*1.0
		}
		compressed[i] = sample * gain
	}
	return compressed
}

func amplitudeEnvelope(env []float64, position float64) float64 {
	if len(env) < 4 {
		return 1.0
	}
	if position <= env[0] {
		return env[1]
	}
	if position >= env[len(env)-2] {
		return env[len(env)-1]
	}
	var t float64
	for i := 0; i < len(env)-2; i += 2 {
		if position >= env[i] && position <= env[i+2] {
			t = (position - env[i]) / (env[i+2] - env[i])
			return (1-t)*env[i+1] + t*env[i+3]
		}
	}
	return 1.0
}

func Envelope(samples []float64, attack, decay, sustainLevel, release float64, sampleRate int) []float64 {
	enveloped := make([]float64, len(samples))
	state := "attack"
	currentLevel := 0.0
	attackSamples := int(attack * float64(sampleRate))
	decaySamples := int(decay * float64(sampleRate))
	releaseSamples := int(release * float64(sampleRate))
	sustainSamples := len(samples) - attackSamples - decaySamples - releaseSamples
	if sustainSamples < 0 {
		sustainSamples = 0
		releaseSamples = len(samples) - attackSamples - decaySamples
		if releaseSamples < 0 {
			releaseSamples = 0
		}
	}
	for i := 0; i < len(samples); i++ {
		switch state {
		case "attack":
			if attackSamples > 0 {
				currentLevel = float64(i) / float64(attackSamples)
				if i >= attackSamples {
					currentLevel = 1.0
					state = "decay"
				}
			} else {
				currentLevel = 1.0
				state = "decay"
			}
		case "decay":
			if decaySamples > 0 {
				currentLevel = 1.0 - ((1.0 - sustainLevel) * float64(i-attackSamples) / float64(decaySamples))
				if i >= attackSamples+decaySamples {
					currentLevel = sustainLevel
					state = "sustain"
				}
			} else {
				currentLevel = sustainLevel
				state = "sustain"
			}
		case "sustain":
			currentLevel = sustainLevel
			if i >= attackSamples+decaySamples+sustainSamples {
				state = "release"
			}
		case "release":
			if releaseSamples > 0 {
				currentLevel = sustainLevel * (1.0 - float64(i-attackSamples-decaySamples-sustainSamples)/float64(releaseSamples))
				if i >= attackSamples+decaySamples+sustainSamples+releaseSamples {
					currentLevel = 0.0
				}
			} else {
				currentLevel = 0.0
			}
		}
		enveloped[i] = samples[i] * currentLevel
	}
	return enveloped
}

func Panning(samples []float64, pan float64) ([]float64, []float64) {
	left := make([]float64, len(samples))
	right := make([]float64, len(samples))
	if pan < -1.0 {
		pan = -1.0
	} else if pan > 1.0 {
		pan = 1.0
	}
	theta := (pan + 1.0) * (math.Pi / 4)
	leftCoeff := math.Cos(theta)
	rightCoeff := math.Sin(theta)
	for i, sample := range samples {
		left[i] = sample * leftCoeff
		right[i] = sample * rightCoeff
	}
	return left, right
}

func Tremolo(samples []float64, sampleRate int, rate, depth float64) []float64 {
	modulated := make([]float64, len(samples))
	lfoPhase := 0.0
	lfoIncrement := rate / float64(sampleRate)
	for i, sample := range samples {
		modulated[i] = sample * (1.0 - depth + depth*math.Sin(2*math.Pi*lfoPhase))
		lfoPhase += lfoIncrement
		if lfoPhase >= 1.0 {
			lfoPhase -= 1.0
		}
	}
	return modulated
}

func Flanger(samples []float64, sampleRate int, baseDelay, modDepth, modRate, feedback, mix float64) []float64 {
	flanged := make([]float64, len(samples))
	bufferSize := int((baseDelay+modDepth)*float64(sampleRate)) + 2
	buffer := make([]float64, bufferSize)
	bufferIndex := 0
	lfoPhase := 0.0
	lfoIncrement := modRate / float64(sampleRate)
	for i, sample := range samples {
		lfoValue := math.Sin(2 * math.Pi * lfoPhase)
		currentDelay := baseDelay + modDepth*lfoValue
		delaySamples := int(currentDelay * float64(sampleRate))
		readIndex := (bufferIndex - delaySamples + bufferSize) % bufferSize
		delayed := buffer[readIndex]
		flanged[i] = sample*(1.0-mix) + delayed*mix
		buffer[bufferIndex] = sample + delayed*feedback
		bufferIndex = (bufferIndex + 1) % bufferSize
		lfoPhase += lfoIncrement
		if lfoPhase >= 1.0 {
			lfoPhase -= 1.0
		}
	}
	return flanged
}

func Phaser(samples []float64, sampleRate int, rate, depth, feedback float64) []float64 {
	phased := make([]float64, len(samples))
	phaseIncrement := rate / float64(sampleRate)
	phase := 0.0
	a0, a1, a2, b1, b2 := lowPassCoefficients("all-pass", 1000.0, 0.7, float64(sampleRate))
	a0a, a1a, a2a, b1a, b2a := lowPassCoefficients("all-pass", 1500.0, 0.7, float64(sampleRate))
	x1, x2, y1, y2 := 0.0, 0.0, 0.0, 0.0
	x1a, x2a, y1a, y2a := 0.0, 0.0, 0.0, 0.0
	for i, sample := range samples {
		sweep := math.Sin(2 * math.Pi * phase)
		centerFreq := 1000.0 + depth*1000.0*sweep
		if centerFreq < 20.0 {
			centerFreq = 20.0
		} else if centerFreq > float64(sampleRate)/2 {
			centerFreq = float64(sampleRate) / 2
		}
		a0, a1, a2, b1, b2 = lowPassCoefficients("all-pass", centerFreq, 0.7, float64(sampleRate))
		a0a, a1a, a2a, b1a, b2a = lowPassCoefficients("all-pass", centerFreq, 0.7, float64(sampleRate))

		var prevPhased float64
		if i > 0 {
			prevPhased = phased[i-1]
		} else {
			prevPhased = 0.0
		}
		gain := feedback * prevPhased

		// First All-Pass Filter
		y := a0*sample + a1*x1 + a2*x2 - b1*y1 - b2*y2
		x2 = x1
		x1 = sample
		y2 = y1
		y1 = y

		// Second All-Pass Filter
		ya := a0a*y + a1a*x1a + a2a*x2a - b1a*y1a - b2a*y2a
		x2a = x1a
		x1a = y
		y2a = y1a
		y1a = ya

		phased[i] = ya + gain
		phase += phaseIncrement
		if phase >= 1.0 {
			phase -= 1.0
		}
	}
	return phased
}

func RingModulation(samples []float64, carrierFreq float64, sampleRate int) []float64 {
	modulated := make([]float64, len(samples))
	carrierPhase := 0.0
	carrierIncrement := carrierFreq / float64(sampleRate)
	for i, sample := range samples {
		carrier := math.Sin(2 * math.Pi * carrierPhase)
		modulated[i] = sample * carrier
		carrierPhase += carrierIncrement
		if carrierPhase >= 1.0 {
			carrierPhase -= 1.0
		}
	}
	return modulated
}

func WahWah(samples []float64, sampleRate int, baseFreq, sweepFreq, Q float64) []float64 {
	phased := make([]float64, len(samples))
	filterType := "band-pass"
	for i, sample := range samples {
		sweep := math.Sin(2 * math.Pi * float64(i) * sweepFreq / float64(sampleRate))
		centerFreq := baseFreq + 500.0*sweep
		if centerFreq < 20.0 {
			centerFreq = 20.0
		} else if centerFreq > float64(sampleRate)/2 {
			centerFreq = float64(sampleRate) / 2
		}
		a0, a1, a2, b1, b2 := lowPassCoefficients(filterType, centerFreq, Q, float64(sampleRate))
		y := a0*sample + a1*0 + a2*0 - b1*0 - b2*0
		phased[i] = y
	}
	return phased
}

func StereoWidening(left, right []float64, width float64) ([]float64, []float64) {
	widenedLeft := make([]float64, len(left))
	widenedRight := make([]float64, len(right))
	for i := 0; i < len(left); i++ {
		widenedLeft[i] = left[i] * (1.0 + width)
		widenedRight[i] = right[i] * (1.0 + width)
		if widenedLeft[i] > 1.0 {
			widenedLeft[i] = 1.0
		} else if widenedLeft[i] < -1.0 {
			widenedLeft[i] = -1.0
		}
		if widenedRight[i] > 1.0 {
			widenedRight[i] = 1.0
		} else if widenedRight[i] < -1.0 {
			widenedRight[i] = -1.0
		}
	}
	return widenedLeft, widenedRight
}

func MultibandCompression(samples []float64, bands []struct {
	Low  float64
	High float64
}, compressors []struct {
	Threshold float64
	Ratio     float64
	Attack    float64
	Release   float64
}, sampleRate int) []float64 {
	if len(bands) != len(compressors) {
		return samples
	}
	splitBands := make([][]float64, len(bands))
	for i, band := range bands {
		splitBands[i] = BandPassFilter(samples, band.Low, band.High, sampleRate)
	}
	for i, compressor := range compressors {
		splitBands[i] = Compressor(splitBands[i], compressor.Threshold, compressor.Ratio, compressor.Attack, compressor.Release, sampleRate)
	}
	recombined := make([]float64, len(samples))
	for _, band := range splitBands {
		for i := range recombined {
			recombined[i] += band[i]
		}
	}
	max := 0.0
	for _, sample := range recombined {
		if math.Abs(sample) > max {
			max = math.Abs(sample)
		}
	}
	if max > 1.0 {
		for i := range recombined {
			recombined[i] /= max
		}
	}
	return recombined
}

func PitchShift(samples []float64, semitones float64) []float64 {
	rate := math.Pow(2, semitones/12)
	newLength := int(float64(len(samples)) / rate)
	shifted := make([]float64, newLength)
	for i := 0; i < newLength; i++ {
		srcIndex := float64(i) * rate
		lower := int(math.Floor(srcIndex))
		upper := lower + 1
		if upper >= len(samples) {
			upper = len(samples) - 1
		}
		frac := srcIndex - float64(lower)
		shifted[i] = samples[lower]*(1-frac) + samples[upper]*frac
	}
	return shifted
}

func FrequencyModulation(samples []float64, carrierFreq, modDepth, sampleRate float64) []float64 {
	modulated := make([]float64, len(samples))
	carrierPhase := 0.0
	for i, sample := range samples {
		instantFreq := carrierFreq + modDepth*sample
		carrierPhase += (instantFreq / sampleRate) * 2 * math.Pi
		if carrierPhase > 2*math.Pi {
			carrierPhase -= 2 * math.Pi
		} else if carrierPhase < 0 {
			carrierPhase += 2 * math.Pi
		}
		modulated[i] = math.Sin(carrierPhase)
	}
	return modulated
}

func PitchModulation(samples []float64, modFreq, modDepth float64, sampleRate int) []float64 {
	modulated := make([]float64, len(samples))
	lfoPhase := 0.0
	lfoIncrement := modFreq / float64(sampleRate)
	maxDelay := int(modDepth * float64(sampleRate))
	buffer := make([]float64, maxDelay*2)
	bufferIndex := 0
	for i := 0; i < len(samples); i++ {
		lfoValue := math.Sin(2 * math.Pi * lfoPhase)
		currentDelay := int(math.Abs(lfoValue) * float64(maxDelay))
		readIndex := (bufferIndex - currentDelay + len(buffer)) % len(buffer)
		modulated[i] = buffer[readIndex]
		buffer[bufferIndex] = samples[i]
		bufferIndex = (bufferIndex + 1) % len(buffer)
		lfoPhase += lfoIncrement
		if lfoPhase >= 1.0 {
			lfoPhase -= 1.0
		}
	}
	return modulated
}

func Reverb(samples []float64, sampleRate int, delayTimes []float64, decays []float64, mix float64) []float64 {
	if len(delayTimes) != len(decays) {
		return samples
	}
	reverb := make([]float64, len(samples))
	numDelays := len(delayTimes)
	buffers := make([][]float64, numDelays)
	bufferIndices := make([]int, numDelays)
	for i := 0; i < numDelays; i++ {
		delaySamples := int(delayTimes[i] * float64(sampleRate))
		if delaySamples <= 0 {
			delaySamples = 1
		}
		buffers[i] = make([]float64, delaySamples)
		bufferIndices[i] = 0
	}
	for i, sample := range samples {
		var wet float64 = 0.0
		for j := 0; j < numDelays; j++ {
			delayed := buffers[j][bufferIndices[j]]
			wet += delayed * decays[j]
			// Update the buffer with the current sample plus feedback
			buffers[j][bufferIndices[j]] = sample + (delayed * decays[j])
			bufferIndices[j] = (bufferIndices[j] + 1) % len(buffers[j])
		}
		// Mix dry and wet signals based on the mix parameter
		reverb[i] = sample*(1.0-mix) + wet*mix
		// Clamp to [-1.0, 1.0] to prevent clipping
		if reverb[i] > 1.0 {
			reverb[i] = 1.0
		} else if reverb[i] < -1.0 {
			reverb[i] = -1.0
		}
	}
	return reverb
}

func Chorus(samples []float64, sampleRate int, delay, depth, rate, mix float64) []float64 {
	chorused := make([]float64, len(samples))
	bufferSize := int((delay+depth)*float64(sampleRate)) + 2
	buffer := make([]float64, bufferSize)
	bufferIndex := 0
	lfoPhase := 0.0
	lfoIncrement := rate / float64(sampleRate)
	for i, sample := range samples {
		lfoValue := math.Sin(2 * math.Pi * lfoPhase)
		currentDelay := delay + depth*lfoValue
		delaySamples := int(currentDelay * float64(sampleRate))
		readIndex := (bufferIndex - delaySamples + bufferSize) % bufferSize
		delayed := buffer[readIndex]
		buffer[bufferIndex] = sample + delayed*0.5
		chorused[i] = sample*(1.0-mix) + delayed*mix
		bufferIndex = (bufferIndex + 1) % bufferSize
		lfoPhase += lfoIncrement
		if lfoPhase >= 1.0 {
			lfoPhase -= 1.0
		}
	}
	return chorused
}

// Bitcrusher applies bit depth and sample rate reduction to the audio samples.
func Bitcrusher(samples []float64, bitDepth int, sampleRateReduction int) []float64 {
	if bitDepth < 1 {
		bitDepth = 1
	}
	if bitDepth > 16 {
		bitDepth = 16
	}
	if sampleRateReduction < 1 {
		sampleRateReduction = 1
	}
	step := 1.0 / math.Pow(2, float64(bitDepth))
	bitcrushed := make([]float64, len(samples))
	var currentSample float64
	for i := 0; i < len(samples); i++ {
		// Apply bit depth reduction (quantization)
		quantized := math.Round(samples[i]/step) * step
		// Apply sample rate reduction by holding the current sample
		if i%sampleRateReduction == 0 {
			currentSample = quantized
		}
		bitcrushed[i] = currentSample
		if bitcrushed[i] > 1.0 {
			bitcrushed[i] = 1.0
		} else if bitcrushed[i] < -1.0 {
			bitcrushed[i] = -1.0
		}
	}
	return bitcrushed
}

func Drive(sample, drive float64) float64 {
	if drive > 0 {
		return sample * (1 + drive) / (1 + drive*math.Abs(sample))
	}
	return sample
}

func Limiter(samples []float64) []float64 {
	limited := make([]float64, len(samples))
	for i, sample := range samples {
		if sample > 1.0 {
			limited[i] = 1.0
		} else if sample < -1.0 {
			limited[i] = -1.0
		} else {
			limited[i] = sample
		}
	}
	return limited
}

func NormalizeSamples(samples []float64, targetPeak float64) []float64 {
	if targetPeak <= 0 {
		return samples
	}
	currentPeak := 0.0
	for _, sample := range samples {
		abs := math.Abs(sample)
		if abs > currentPeak {
			currentPeak = abs
		}
	}
	if currentPeak == 0 {
		return samples
	}
	scale := targetPeak / currentPeak
	normalizedSamples := make([]float64, len(samples))
	for i, sample := range samples {
		normalizedSamples[i] = sample * scale
		if normalizedSamples[i] > 1.0 {
			normalizedSamples[i] = 1.0
		} else if normalizedSamples[i] < -1.0 {
			normalizedSamples[i] = -1.0
		}
	}
	return normalizedSamples
}

func SubtractOp(duration, amplitude float64, b1 float64, ampEnv []float64, sampleRate int) []float64 {
	samples := make([]float64, int(duration*float64(sampleRate)))
	for i := 0; i < len(samples); i++ {
		noise := rand.Float64()*2 - 1
		samples[i] = b1 * noise
		samples[i] *= amplitudeEnvelope(ampEnv, float64(i)/float64(len(samples)))
	}
	return samples
}

func AddPartials(duration, amplitude, frequency float64, partials []float64, ampEnv []float64, sampleRate int) []float64 {
	samples := make([]float64, int(duration*float64(sampleRate)))
	for i := 0; i < len(samples); i++ {
		t := float64(i) / float64(sampleRate)
		for j := 0; j < len(partials); j += 2 {
			partialsFreq := partials[j] * frequency
			partialsAmp := partials[j+1]
			samples[i] += partialsAmp * math.Sin(2*math.Pi*partialsFreq*t)
		}
		samples[i] *= amplitudeEnvelope(ampEnv, float64(i)/float64(len(samples)))
		samples[i] *= amplitude
	}
	return samples
}

func FMSynthesis(duration, carrierFreq, modFreq, modIndex, amplitude float64, ampEnv []float64, sampleRate int) []float64 {
	samples := make([]float64, int(duration*float64(sampleRate)))
	for i := 0; i < len(samples); i++ {
		t := float64(i) / float64(sampleRate)
		modulator := modIndex * math.Sin(2*math.Pi*modFreq*t)
		samples[i] = amplitude * math.Sin(2*math.Pi*carrierFreq*t+modulator) * amplitudeEnvelope(ampEnv, float64(i)/float64(len(samples)))
	}
	return samples
}

func KarplusStrong(duration, amplitude float64, p int, b float64, sampleRate int) []float64 {
	samples := make([]float64, int(duration*float64(sampleRate)))
	buffer := make([]float64, p)
	for i := range buffer {
		buffer[i] = rand.Float64()*2 - 1
	}
	for i := 0; i < len(samples); i++ {
		samples[i] = buffer[i%p] * amplitude
		average := 0.5 * (buffer[i%p] + buffer[(i+1)%p])
		buffer[i%p] = b*average + (1-b)*buffer[i%p]
	}
	return samples
}

func GranularSynthesis(samples []float64, grainSize, overlap int, sampleRate int) []float64 {
	output := make([]float64, len(samples))
	numGrains := len(samples) / grainSize
	grains := make([][]float64, numGrains)
	for i := 0; i < numGrains; i++ {
		grains[i] = make([]float64, grainSize)
		copy(grains[i], samples[i*grainSize:(i+1)*grainSize])
	}
	for _, grain := range grains {
		for j := 0; j < grainSize; j++ {
			output[j] += grain[j]
		}
	}
	return output
}

// QuadraticFadeIn applies a quadratic fade-in to the provided samples.
// The fade-in starts slowly and accelerates towards the end of the duration.
func QuadraticFadeIn(samples []float64, duration float64, sampleRate int) []float64 {
	faded := make([]float64, len(samples))
	fadeSamples := int(duration * float64(sampleRate))
	if fadeSamples > len(samples) {
		fadeSamples = len(samples)
	}
	var t float64
	for i := 0; i < len(samples); i++ {
		if i < fadeSamples {
			t = float64(i) / float64(fadeSamples)
			faded[i] = samples[i] * t * t
		} else {
			faded[i] = samples[i]
		}
	}
	return faded
}

// QuadraticFadeOut applies a quadratic fade-out to the provided samples.
// The fade-out starts quickly and decelerates towards the end of the duration.
func QuadraticFadeOut(samples []float64, duration float64, sampleRate int) []float64 {
	faded := make([]float64, len(samples))
	fadeSamples := int(duration * float64(sampleRate))
	startFade := len(samples) - fadeSamples
	if startFade < 0 {
		startFade = 0
	}
	var t float64
	for i := 0; i < len(samples); i++ {
		if i >= startFade {
			t = float64(len(samples)-i) / float64(fadeSamples)
			faded[i] = samples[i] * t * t
		} else {
			faded[i] = samples[i]
		}
	}
	return faded
}

// EnvelopeAtTime calculates the envelope value at a specific normalized time point.
func EnvelopeAtTime(t, attack, decay, sustainLevel, release, duration float64) float64 {
	if t < 0 {
		return 0.0
	}
	if t < attack {
		return t / attack
	}
	if t < attack+decay {
		return 1.0 - ((t-attack)/decay)*(1.0-sustainLevel)
	}
	if t < duration-release {
		return sustainLevel
	}
	if t < duration {
		return sustainLevel * (1.0 - (t-(duration-release))/release)
	}
	return 0.0
}

// Shimmer applies a shimmer effect with adjustable pitch shift and feedback to the provided audio samples.
// It adds a delayed and pitch-shifted copy of the signal to itself with optional feedback.
func Shimmer(samples []float64, sampleRate int, delayTime float64, mix float64, pitchShiftSemitones float64, feedback float64) []float64 {
	if delayTime < 0 {
		delayTime = 0
	}
	if mix < 0 {
		mix = 0
	} else if mix > 1.0 {
		mix = 1.0
	}
	if feedback < 0.0 {
		feedback = 0.0
	} else if feedback >= 1.0 {
		feedback = 0.99 // Prevent infinite feedback
	}

	// Calculate the number of samples to delay
	delaySamples := int(delayTime * float64(sampleRate))
	if delaySamples >= len(samples) {
		delaySamples = len(samples) - 1
	}

	// Pitch shift by the specified number of semitones
	pitchShiftFactor := math.Pow(2, pitchShiftSemitones/12.0)

	// Create the pitch-shifted copy
	pitchShifted := pitchShift(samples, pitchShiftFactor)

	// Prepare the shimmer signal with delay and feedback
	shimmer := make([]float64, len(samples))
	for i := 0; i < len(shimmer); i++ {
		if i >= delaySamples && (i-delaySamples) < len(pitchShifted) {
			shimmer[i] = pitchShifted[i-delaySamples] + shimmer[i-delaySamples]*feedback
		} else {
			shimmer[i] = 0.0
		}
	}

	// Mix the shimmer signal with the original signal
	output := make([]float64, len(samples))
	for i := 0; i < len(samples); i++ {
		output[i] = samples[i]*(1.0-mix) + shimmer[i]*mix

		// Clamp the output to [-1.0, 1.0] to prevent clipping
		if output[i] > 1.0 {
			output[i] = 1.0
		} else if output[i] < -1.0 {
			output[i] = -1.0
		}
	}

	return output
}

// pitchShift shifts the pitch of the audio samples by the given factor.
// A factor >1.0 shifts the pitch up, while a factor <1.0 shifts it down.
func pitchShift(samples []float64, factor float64) []float64 {
	if factor <= 0 {
		return samples
	}

	originalLength := len(samples)
	newLength := int(float64(originalLength) / factor)
	pitched := make([]float64, newLength)

	for i := 0; i < newLength; i++ {
		origIndex := float64(i) * factor
		indexFloor := int(math.Floor(origIndex))
		indexCeil := indexFloor + 1
		if indexCeil >= originalLength {
			indexCeil = originalLength - 1
		}
		weight := origIndex - float64(indexFloor)
		pitched[i] = samples[indexFloor]*(1.0-weight) + samples[indexCeil]*weight
	}

	return pitched
}

// ShimmerBitcrusher applies both Shimmer and Bitcrusher effects to the provided audio samples.
// The Shimmer effect is applied first, followed by the Bitcrusher effect.
//
// Parameters:
// - samples: The input audio samples to be processed.
// - sampleRate: The sample rate of the audio in Hz.
// - delayTime: The delay time in seconds before the shimmer is mixed back.
// - mix: The mix level of the shimmer effect (0.0 to 1.0).
// - pitchShiftSemitones: The number of semitones to shift the pitch for the shimmer.
// - bitDepth: The number of bits to reduce the sample precision in bitcrushing.
// - sampleRateReduction: The number of samples to hold each sample value in bitcrushing.
//
// Returns:
// - A new slice of samples with both Shimmer and Bitcrusher effects applied.
func ShimmerBitcrusher(samples []float64, sampleRate int, delayTime float64, mix float64, pitchShiftSemitones float64, bitDepth int, sampleRateReduction int, feedback float64) []float64 {
	return Bitcrusher(Shimmer(samples, sampleRate, delayTime, mix, pitchShiftSemitones, feedback), bitDepth, sampleRateReduction)
}
