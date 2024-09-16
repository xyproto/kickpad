package synth

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// SaveToWav saves the waveform to a .wav file using int16 PCM format.
func SaveToWav(filename string, samples []float64, sampleRate int) error {
	if len(samples) == 0 {
		return fmt.Errorf("cannot save empty waveform: no samples provided")
	}

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating wav file: %v", err)
	}
	defer outFile.Close()

	// Create a new WAV encoder for int16 PCM
	enc := wav.NewEncoder(outFile, sampleRate, 16, 1, 1) // 16-bit, mono channel

	// Create an IntBuffer to store the int16 PCM data
	buf := &audio.IntBuffer{
		Format: &audio.Format{SampleRate: sampleRate, NumChannels: 1},
		Data:   make([]int, len(samples)), // Store int16 samples as int
	}

	// Convert from float64 to int16
	for i, sample := range samples {
		scaled := sample * float64(math.MaxInt16)                                                     // Scale to int16 range
		buf.Data[i] = int(math.Max(math.Min(scaled, float64(math.MaxInt16)), float64(math.MinInt16))) // Clamp to int16
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

// PlayWav plays a WAV file using mpv or ffmpeg
func PlayWav(filePath string) error {
	cmd := exec.Command("mpv", filePath)
	err := cmd.Start()
	if err != nil {
		// Fallback to ffmpeg if mpv is not available
		cmd = exec.Command("ffmpeg", "-i", filePath, "-f", "null", "-")
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("error playing sound with both mpv and ffmpeg: %v", err)
		}
	}
	cmd.Wait()
	return nil
}

// Play plays the generated kick sound by writing it to a temporary WAV file and playing it with an external player
func (cfg *Settings) Play() error {
	// Generate the kick waveform in memory
	samples, err := cfg.GenerateKickWaveform()
	if err != nil {
		return err
	}

	// Save the waveform to a temporary WAV file
	tmpFile, err := os.CreateTemp("", "kick_*.wav")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	err = SaveToWav(tmpFile.Name(), samples, cfg.SampleRate)
	if err != nil {
		return err
	}

	// Play the generated WAV file using an external player (mpv or ffmpeg)
	return PlayWav(tmpFile.Name())
}

// SaveTo saves the generated kick to a specified directory, avoiding filename collisions.
func (cfg *Settings) SaveTo(directory string) (string, error) {
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
