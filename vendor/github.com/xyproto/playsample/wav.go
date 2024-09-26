package playsample

import (
	"fmt"
	"io"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

func SaveToWav(w io.WriteSeeker, samples []float64, sampleRate, bitDepth, channels int) error {
	if len(samples) == 0 {
		return fmt.Errorf("cannot save empty waveform: no samples provided")
	}
	if bitDepth == 0 {
		return fmt.Errorf("bitdepth should be 8, 16, 24 or 32, not 0")
	}
	const audioFormat = 1
	enc := wav.NewEncoder(w, sampleRate, bitDepth, channels, audioFormat)
	buf := &audio.IntBuffer{
		Format: &audio.Format{SampleRate: sampleRate, NumChannels: 1},
		Data:   make([]int, len(samples)),
	}
	switch bitDepth {
	case 8:
		// Convert from float64 to int8 for 8-bit WAV files
		for i, sample := range samples {
			scaled := sample*127 + 128 // Scale to unsigned 8-bit range (0 to 255)
			buf.Data[i] = int(math.Max(math.Min(scaled, 255), 0))
		}
	case 16:
		// Convert from float64 to int16 for 16-bit WAV files
		for i, sample := range samples {
			scaled := sample * float64(math.MaxInt16) // Scale to int16 range
			buf.Data[i] = int(math.Max(math.Min(scaled, float64(math.MaxInt16)), float64(math.MinInt16)))
		}
	case 24:
		// Convert from float64 to int24 for 24-bit WAV files
		for i, sample := range samples {
			scaled := sample * float64(1<<23) // Scale to 24-bit range (-8388608 to 8388607)
			buf.Data[i] = int(math.Max(math.Min(scaled, float64(8388607)), float64(-8388608)))
		}
	case 32:
		// Convert from float64 to int32 for 32-bit WAV files
		for i, sample := range samples {
			scaled := sample * float64(math.MaxInt32) // Scale to int32 range
			buf.Data[i] = int(math.Max(math.Min(scaled, float64(math.MaxInt32)), float64(math.MinInt32)))
		}
	default:
		return fmt.Errorf("unsupported bit depth: %d", bitDepth)
	}
	if err := enc.Write(buf); err != nil {
		return fmt.Errorf("error writing wav file: %v", err)
	}
	return enc.Close()
}

// LoadWav loads a WAV file and converts mono to stereo if the "monoToStereo" flag is true.
// It returns the samples as []float64 and the sample rate.
func LoadWav(filename string, monoToStereo bool) ([]float64, int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening wav file: %v", err)
	}
	defer f.Close()
	decoder := wav.NewDecoder(f)
	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("error decoding wav file: %v", err)
	}
	intBuffer := buffer.Data
	numSamples := len(intBuffer)
	sampleRate := buffer.Format.SampleRate
	numChannels := buffer.Format.NumChannels
	// If mono and monoToStereo is true, duplicate samples to stereo
	if numChannels == 1 && monoToStereo {
		stereoSamples := make([]float64, numSamples*2)
		for i := 0; i < numSamples; i++ {
			monoSample := float64(intBuffer[i]) / math.MaxInt16
			stereoSamples[2*i] = monoSample   // Left channel
			stereoSamples[2*i+1] = monoSample // Right channel
		}
		return stereoSamples, sampleRate, nil
	}
	// If stereo or if monoToStereo is false, convert to []float64 directly
	samples := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = float64(intBuffer[i]) / math.MaxInt16
	}
	return samples, sampleRate, nil
}
