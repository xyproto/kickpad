package synth

import (
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// SaveToWav saves the waveform to a .wav file using the PCM format with the given bit depth.
func SaveToWav(w io.WriteSeeker, samples []float64, sampleRate, bitDepth int) error {
	if len(samples) == 0 {
		return fmt.Errorf("cannot save empty waveform: no samples provided")
	}

	// Create a new WAV encoder for intN PCM
	enc := wav.NewEncoder(w, sampleRate, bitDepth, 1, 1) // Mono channel

	// Create a buffer to store the PCM data
	buf := &audio.IntBuffer{
		Format: &audio.Format{SampleRate: sampleRate, NumChannels: 1},
		Data:   make([]int, len(samples)),
	}

	switch bitDepth {
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
	default:
		return fmt.Errorf("unsupported bit depth: %d", bitDepth)
	}

	// Write the IntBuffer to the WAV file
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

// FFPlayWav plays a WAV file using ffplay
func FFPlayWav(filePath string) error {
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", filePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error playing sound with ffplay: %v", err)
	}
	return cmd.Wait()
}

// FFPlayKick generates a Kick drum waveform, saves it to a temporary WAV file,
// plays it using ffplay, and then deletes the temporary file.
func FFPlayKick(cfg *Settings) error {
	// Generate the kick drum waveform
	samples, err := cfg.GenerateKickWaveform()
	if err != nil {
		return fmt.Errorf("error generating kick waveform: %v", err)
	}

	// Create a temporary WAV file
	tmpFile, err := os.CreateTemp("", "kickdrum_*.wav")
	if err != nil {
		return fmt.Errorf("error creating temporary wav file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Ensure the file is removed after playing

	// Save the waveform to the temporary file using SaveToWav
	if err := SaveToWav(tmpFile, samples, cfg.SampleRate, cfg.BitDepth); err != nil {
		return fmt.Errorf("error saving wav file: %v", err)
	}

	// Play the temporary wav file using ffplay
	err = FFPlayWav(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("error playing wav file: %v", err)
	}

	return nil
}

// SaveKickTo generates kick samples and saves it to a specified directory, avoiding filename collisions.
func (cfg *Settings) SaveKickTo(directory string) (string, error) {
	n := 1
	var fileName string
	for {
		// Construct the file path with an incrementing number
		fileName = filepath.Join(directory, fmt.Sprintf("kick%d.wav", n))
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			break
		}
		n++
	}

	// Create the new file
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Set the file as the output for the kick generation
	cfg.Output = file

	// Generate the kick and write to the file
	if err := cfg.GenerateKick(); err != nil {
		return "", err
	}

	return fileName, nil
}
