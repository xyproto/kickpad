# Audio Effects

Apply audio effects to samples.

Note that this package is a bit experimental and a work in progress.

Audio samples are passed in as `[]float64`. The sample rate is typically 44100 or 48000.

### Example use

Write a .wav file that has a very simple clap-like sound, with some reverb applied:

```go
package main

import (
    "log"
    "math"
    "math/rand"
    "os"

    "github.com/go-audio/audio"
    "github.com/go-audio/wav"
    "github.com/xyproto/audioeffects"
)

func writeWav(filename string, samples []float64, sampleRate int, numChannels int) error {
    var (
        ch, intSample int
        intSamples    = make([]int, len(samples)*numChannels)
    )
    for i, sample := range samples {
        if sample > 1.0 {
            sample = 1.0
        } else if sample < -1.0 {
            sample = -1.0
        }
        intSample = int(math.Round(sample * 32767))
        for ch = 0; ch < numChannels; ch++ {
            intSamples[i*numChannels+ch] = intSample
        }
    }
    buf := &audio.IntBuffer{
        Data:           intSamples,
        Format:         &audio.Format{SampleRate: sampleRate, NumChannels: numChannels},
        SourceBitDepth: 16,
    }
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    enc := wav.NewEncoder(file, sampleRate, 16, numChannels, 1)
    if err := enc.Write(buf); err != nil {
        return err
    }
    return enc.Close()
}

func main() {
    sampleRate := 48000
    duration := 3.0

    numSamples := int(duration * float64(sampleRate))
    samples := make([]float64, numSamples)

    // Create white noise burst with quick decay
    for i := 0; i < numSamples; i++ {
        if i < int(float64(sampleRate)*0.05) { // 50 ms of noise
            samples[i] = (rand.Float64()*2 - 1) * (1.0 - float64(i)/(float64(sampleRate)*0.05))
        } else {
            samples[i] = 0.0
        }
    }

    // Apply Reverb with configurable mix
    delayTimes := []float64{0.2, 0.4, 0.6, 0.8} // in seconds
    decays := []float64{0.6, 0.4, 0.2, 0.1}     // decay factors
    mix := 0.3                                  // 30% wet signal, 70% dry signal

    reverbed := audioeffects.Reverb(samples, sampleRate, delayTimes, decays, mix)

    // Normalize to prevent clipping
    normalized := audioeffects.NormalizeSamples(reverbed, 0.8)

    const filename = "clap_reverb.wav"

    err := writeWav(filename, normalized, sampleRate, 1)
    if err != nil {
        log.Fatalf("Failed to write WAV to %s: %v", filename, err)
    }
    log.Printf("Successfully wrote %s\n", filename)
}
```

### Exported functions in this package

```
func lowPassCoefficients(filterType string, freq, Q, sampleRate float64) (float64, float64, float64, float64, float64)
func BiquadFilter(samples []float64, filterType string, freq, Q, sampleRate float64) []float64
func FadeIn(samples []float64, duration float64, sampleRate int) []float64
func FadeOut(samples []float64, duration float64, sampleRate int) []float64
func LowPassFilter(samples []float64, cutoffFreq float64, sampleRate int) []float64
func HighPassFilter(samples []float64, cutoffFreq float64, sampleRate int) []float64
func BandPassFilter(samples []float64, lowFreq, highFreq float64, sampleRate int) []float64
func NoiseGate(samples []float64, threshold, attack, release float64, sampleRate int) []float64
func StereoDelay(left, right []float64, sampleRate int, delayTimeLeft, delayTimeRight float64, feedback, mix float64) ([]float64, []float64)
func Expander(samples []float64, threshold, ratio, attack, release float64, sampleRate int) []float64
func SoftClip(sample, drive float64) float64
func SoftClippingDistortion(samples []float64, drive float64) []float64
func SidechainCompressor(target, trigger []float64, threshold, ratio, attack, release float64, sampleRate int) []float64
func Compressor(samples []float64, threshold, ratio, attack, release float64, sampleRate int) []float64
func Envelope(samples []float64, attack, decay, sustainLevel, release float64, sampleRate int) []float64
func Panning(samples []float64, pan float64) ([]float64, []float64)
func Tremolo(samples []float64, sampleRate int, rate, depth float64) []float64
func Flanger(samples []float64, sampleRate int, baseDelay, modDepth, modRate, feedback, mix float64) []float64
func Phaser(samples []float64, sampleRate int, rate, depth, feedback float64) []float64
func RingModulation(samples []float64, carrierFreq float64, sampleRate int) []float64
func WahWah(samples []float64, sampleRate int, baseFreq, sweepFreq, Q float64) []float64
func StereoWidening(left, right []float64, width float64) ([]float64, []float64)
func MultibandCompression(samples []float64, bands []struct
func PitchShift(samples []float64, semitones float64) []float64
func FrequencyModulation(samples []float64, carrierFreq, modDepth, sampleRate float64) []float64
func PitchModulation(samples []float64, modFreq, modDepth float64, sampleRate int) []float64
func Reverb(samples []float64, sampleRate int, delayTimes []float64, decays []float64, mix float64) []float64
func Chorus(samples []float64, sampleRate int, delay, depth, rate, mix float64) []float64
func Bitcrusher(samples []float64, bitDepth int, sampleRateReduction int) []float64
func Drive(sample, drive float64) float64
func Limiter(samples []float64) []float64
func NormalizeSamples(samples []float64, targetPeak float64) []float64
func SubtractOp(duration, amplitude float64, b1 float64, ampEnv []float64, sampleRate int) []float64
func AddPartials(duration, amplitude, frequency float64, partials []float64, ampEnv []float64, sampleRate int) []float64
func FMSynthesis(duration, carrierFreq, modFreq, modIndex, amplitude float64, ampEnv []float64, sampleRate int) []float64
func KarplusStrong(duration, amplitude float64, p int, b float64, sampleRate int) []float64
func GranularSynthesis(samples []float64, grainSize, overlap int, sampleRate int) []float64
func QuadraticFadeIn(samples []float64, duration float64, sampleRate int) []float64
func QuadraticFadeOut(samples []float64, duration float64, sampleRate int) []float64
func EnvelopeAtTime(t, attack, decay, sustainLevel, release, duration float64) float64
func Shimmer(samples []float64, sampleRate int, delayTime float64, mix float64, pitchShiftSemitones float64, feedback float64) []float64
func ShimmerBitcrusher(samples []float64, sampleRate int, delayTime float64, mix float64, pitchShiftSemitones float64, bitDepth int, sampleRateReduction int, feedback float64) []float64
```

### General info

* Version: 0.11.1
* License: MIT
