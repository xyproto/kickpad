package synth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xyproto/playsample"
)

// GenerateAndSaveTo generates samples for a given type (e.g., "kick", "snare") and saves it to a specified directory, avoiding filename collisions.
func (cfg *Settings) GenerateAndSaveTo(t, directory string) (string, error) {
	n := 1
	var fileName string
	for {
		// Construct the file path with an incrementing number based on the type
		fileName = filepath.Join(directory, fmt.Sprintf("%s%d.wav", t, n))
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
	// Set the file as the output for the sound generation
	cfg.Output = file
	// Generate the samples for the requested type
	samples, err := cfg.Generate(t)
	if err != nil {
		return "", err
	}
	// Save the generated samples to the WAV file
	err = playsample.SaveToWav(file, samples, cfg.SampleRate, cfg.BitDepth, cfg.Channels)
	if err != nil {
		return "", fmt.Errorf("error saving to wav file: %v", err)
	}
	return fileName, nil
}
