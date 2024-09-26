package synth

import (
	"github.com/xyproto/audioeffects"
)

// ApplyEnvelope applies an ADSR envelope to the waveform using the audioeffects package.
func ApplyEnvelope(samples []float64, attack, decay, sustain, release float64, sampleRate int) []float64 {
	return audioeffects.Envelope(samples, attack, decay, sustain, release, sampleRate)
}

// ApplyEnvelopeAtTime generates the ADSR envelope value at a specific normalized time.
// This function retains the custom implementation as audioeffects does not expose envelope evaluation at a specific time.
func (cfg *Settings) ApplyEnvelopeAtTime(t float64) float64 {
	return audioeffects.EnvelopeAtTime(t, cfg.Attack, cfg.Decay, cfg.Sustain, cfg.Release, cfg.Duration)
}

// ApplyDrive applies a drive (distortion) effect to a single sample using the audioeffects package.
func (cfg *Settings) ApplyDrive(sample float64) float64 {
	return audioeffects.Drive(sample, cfg.Drive)
}

// ApplyPitchModulation applies pitch modulation (vibrato) to the samples using the audioeffects package.
func ApplyPitchModulation(samples []float64, modFreq, modDepth float64, sampleRate int) []float64 {
	return audioeffects.PitchModulation(samples, modFreq, modDepth, sampleRate)
}

// ApplyPanning applies stereo panning to the samples using the audioeffects package.
// Pan should be in the range [-1, 1], where -1 is full left and 1 is full right.
func ApplyPanning(samples []float64, pan float64) ([]float64, []float64) {
	return audioeffects.Panning(samples, pan)
}

// ApplyFrequencyModulation applies frequency modulation to a waveform using the audioeffects package.
// carrierFreq is the base frequency of the carrier wave.
// modDepth controls the extent of frequency deviation.
func ApplyFrequencyModulation(samples []float64, carrierFreq, modDepth float64, sampleRate int) []float64 {
	return audioeffects.FrequencyModulation(samples, carrierFreq, modDepth, float64(sampleRate))
}

// ApplyFadeIn applies a fade-in to the start of the samples using the audioeffects package.
func ApplyFadeIn(samples []float64, fadeDuration float64, sampleRate int) []float64 {
	return audioeffects.FadeIn(samples, fadeDuration, sampleRate)
}

// ApplyQuadraticFadeIn applies a fade-in to the start of the samples using the audioeffects package.
func ApplyQuadraticFadeIn(samples []float64, fadeDuration float64, sampleRate int) []float64 {
	return audioeffects.QuadraticFadeIn(samples, fadeDuration, sampleRate)
}

// ApplyFadeOut applies a fade-out to the end of the samples using the audioeffects package.
func ApplyFadeOut(samples []float64, fadeDuration float64, sampleRate int) []float64 {
	return audioeffects.FadeOut(samples, fadeDuration, sampleRate)
}

// ApplyQuadraticFadeOut applies a fade-out to the end of the samples using the audioeffects package.
func ApplyQuadraticFadeOut(samples []float64, fadeDuration float64, sampleRate int) []float64 {
	return audioeffects.QuadraticFadeOut(samples, fadeDuration, sampleRate)
}

// ApplyChorus applies a chorus effect to the samples using the audioeffects package.
// delaySec specifies the base delay time in seconds.
// depth controls the modulation depth of the delay time.
// rate is the modulation rate in Hz.
// mix determines the blend between dry and wet signals (0.0 = dry only, 1.0 = wet only).
func ApplyChorus(samples []float64, sampleRate int, delaySec, depth, rate, mix float64) []float64 {
	return audioeffects.Chorus(samples, sampleRate, delaySec, depth, rate, mix)
}

// ApplyReverb applies a reverb effect to the samples using the audioeffects package.
// delayTimes and decays should be of the same length, specifying multiple delay and decay pairs.
// mix determines the blend between dry and wet signals.
func ApplyReverb(samples []float64, sampleRate int, delayTimes, decays []float64, mix float64) []float64 {
	return audioeffects.Reverb(samples, sampleRate, delayTimes, decays, mix)
}

// ApplyCompressor applies dynamic range compression to the samples using the audioeffects package.
// threshold sets the level above which compression occurs.
// ratio determines the amount of compression applied.
// attack and release control the compressor's responsiveness.
func ApplyCompressor(samples []float64, threshold, ratio, attack, release float64, sampleRate int) []float64 {
	return audioeffects.Compressor(samples, threshold, ratio, attack, release, sampleRate)
}

// ApplyLimiter applies a limiter to the samples using the audioeffects package.
// It ensures that the sample amplitudes do not exceed 1.0 or -1.0.
func ApplyLimiter(samples []float64) []float64 {
	return audioeffects.Limiter(samples)
}

// ApplyNormalize scales the samples to ensure the peak amplitude matches the target peak using the audioeffects package.
func ApplyNormalize(samples []float64, targetPeak float64) []float64 {
	return audioeffects.NormalizeSamples(samples, targetPeak)
}

// ApplyStereoDelay applies a stereo delay effect to the left and right channels using the audioeffects package.
// delayTimeLeft and delayTimeRight specify delay times for each channel in seconds.
// feedback controls the amount of delayed signal fed back into the delay line.
// mix determines the blend between dry and wet signals.
func ApplyStereoDelay(left, right []float64, sampleRate int, delayTimeLeft, delayTimeRight, feedback, mix float64) ([]float64, []float64) {
	return audioeffects.StereoDelay(left, right, sampleRate, delayTimeLeft, delayTimeRight, feedback, mix)
}

// ApplyBitcrusher applies a bitcrusher effect to the samples using the audioeffects package.
// bitDepth controls the number of bits used in the reduction.
// sampleRateReduction reduces the sample rate by the specified factor.
func ApplyBitcrusher(samples []float64, bitDepth, sampleRateReduction int) []float64 {
	return audioeffects.Bitcrusher(samples, bitDepth, sampleRateReduction)
}

// ApplySoftClipping applies soft clipping distortion to the samples using the audioeffects package.
func ApplySoftClipping(samples []float64, drive float64) []float64 {
	return audioeffects.SoftClippingDistortion(samples, drive)
}

// ApplyTremolo applies a tremolo effect to the samples using the audioeffects package.
// rate is the modulation frequency in Hz.
// depth controls the amplitude modulation depth.
func ApplyTremolo(samples []float64, sampleRate int, rate, depth float64) []float64 {
	return audioeffects.Tremolo(samples, sampleRate, rate, depth)
}

// ApplyFlanger applies a flanger effect to the samples using the audioeffects package.
// baseDelay is the base delay time in seconds.
// modDepth controls the modulation depth of the delay time.
// modRate is the modulation rate in Hz.
// feedback controls the amount of delayed signal fed back into the delay line.
// mix determines the blend between dry and wet signals.
func ApplyFlanger(samples []float64, sampleRate int, baseDelay, modDepth, modRate, feedback, mix float64) []float64 {
	return audioeffects.Flanger(samples, sampleRate, baseDelay, modDepth, modRate, feedback, mix)
}

// ApplyPhaser applies a phaser effect to the samples using the audioeffects package.
// rate is the modulation frequency in Hz.
// depth controls the modulation depth.
// feedback determines the amount of phase-shifted signal fed back into the phaser.
func ApplyPhaser(samples []float64, sampleRate int, rate, depth, feedback float64) []float64 {
	return audioeffects.Phaser(samples, sampleRate, rate, depth, feedback)
}

// ApplySidechainCompressor applies a sidechain compressor to the target samples using the trigger samples.
// threshold sets the compression threshold.
// ratio determines the compression ratio.
// attack and release control the compressor's responsiveness.
func ApplySidechainCompressor(target, trigger []float64, threshold, ratio, attack, release float64, sampleRate int) []float64 {
	return audioeffects.SidechainCompressor(target, trigger, threshold, ratio, attack, release, sampleRate)
}

// ApplyNoiseGate applies a noise gate to the samples using the audioeffects package.
// threshold sets the level below which the signal is attenuated.
// attack and release control the gate's responsiveness.
func ApplyNoiseGate(samples []float64, threshold, attack, release float64, sampleRate int) []float64 {
	return audioeffects.NoiseGate(samples, threshold, attack, release, sampleRate)
}

// ApplyMultibandCompression applies multiband compression to the samples using the audioeffects package.
// bands defines the frequency ranges for each band.
// compressors defines the compression settings for each band.
func ApplyMultibandCompression(samples []float64, bands []struct {
	Low  float64
	High float64
}, compressors []struct {
	Threshold float64
	Ratio     float64
	Attack    float64
	Release   float64
}, sampleRate int) []float64 {
	return audioeffects.MultibandCompression(samples, bands, compressors, sampleRate)
}

// ApplyGranularSynthesis applies granular synthesis to the samples using the audioeffects package.
// grainSize specifies the size of each grain in samples.
// overlap determines the overlap between consecutive grains.
func ApplyGranularSynthesis(samples []float64, grainSize, overlap, sampleRate int) []float64 {
	return audioeffects.GranularSynthesis(samples, grainSize, overlap, sampleRate)
}

// ApplyKarplusStrong applies the Karplus-Strong algorithm to generate plucked string sounds using the audioeffects package.
// p is the delay line length in samples.
// b controls the damping factor.
func ApplyKarplusStrong(duration, amplitude float64, p int, b float64, sampleRate int) []float64 {
	return audioeffects.KarplusStrong(duration, amplitude, p, b, sampleRate)
}

// ApplyFM_Synthesis generates a frequency-modulated synthesis signal using the audioeffects package.
func ApplyFMSynthesis(duration, carrierFreq, modFreq, modIndex, amplitude float64, ampEnv []float64, sampleRate int) []float64 {
	return audioeffects.FMSynthesis(duration, carrierFreq, modFreq, modIndex, amplitude, ampEnv, sampleRate)
}

// ApplyAddPartials adds harmonic partials to the samples using the audioeffects package.
// partials should contain frequency and amplitude pairs.
func ApplyAddPartials(duration, amplitude, frequency float64, partials, ampEnv []float64, sampleRate int) []float64 {
	return audioeffects.AddPartials(duration, amplitude, frequency, partials, ampEnv, sampleRate)
}

// ApplySubtractOp subtracts noise from the samples using the audioeffects package.
// duration specifies the noise duration in seconds.
// amplitude sets the noise amplitude.
// b1 is a scaling factor for the noise.
// ampEnv defines the amplitude envelope.
func ApplySubtractOp(duration, amplitude, b1 float64, ampEnv []float64, sampleRate int) []float64 {
	return audioeffects.SubtractOp(duration, amplitude, b1, ampEnv, sampleRate)
}
