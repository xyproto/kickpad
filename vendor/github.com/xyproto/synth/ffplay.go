package synth

import (
	"fmt"
	"os"
	"os/exec"
)

// FFPlayWav plays a WAV file using ffplay
func FFPlayWav(filePath string) error {
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", filePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error playing sound with ffplay: %v", err)
	}
	return cmd.Wait()
}

// FFGeneratePlay generates a waveform of the given type, saves it to a temporary WAV file,
// plays it using ffplay, and then deletes the temporary file.
func FFGeneratePlay(t string, cfg *Settings) error {
	// Generate the kick drum waveform
	samples, err := cfg.Generate(t)
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
