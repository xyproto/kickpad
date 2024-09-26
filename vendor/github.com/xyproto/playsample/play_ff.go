//go:build ff

package playsample

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/xyproto/files"
)

// Player represents a struct with methods for playing audio
type Player struct {
	Initialized bool
}

// NewPlayer checks if ffplay exists in the PATH and returns a new Player struct
func NewPlayer() *Player {
	return &Player{Initialized: files.PathHas("ffplay")}
}

// Close sets the Player as uninitialized
func (player *Player) Close() {
	player.Initialized = false
}

// PlayWav plays a WAV file using SDL2 and SDL_mixer
func (player *Player) PlayWav(filePath string) error {
	if !player.Initialized {
		return errors.New("ffplay culd not be found")
	}
	return FFPlayWav(filePath)
}

// PlayWaveform plays raw waveform samples using SDL2
func (player *Player) PlayWaveform(samples []float64, sampleRate, bitDepth, channels int) error {
	if !player.Initialized {
		return errors.New("ffplay culd not be found")
	}
	tmpFile, err := os.CreateTemp("", "waveform_*.wav")
	if err != nil {
		return fmt.Errorf("error creating temporary wav file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if err := SaveToWav(tmpFile, samples, sampleRate, bitDepth, channels); err != nil {
		return fmt.Errorf("error saving wav file: %v", err)
	}
	err = FFPlayWavWithSampleRate(tmpFile.Name(), sampleRate)
	if err != nil {
		return fmt.Errorf("error playing wav file: %v", err)
	}
	return nil
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

// FFPlayWav plays a WAV file using ffplay and by specifying a sample rate
func FFPlayWavWithSampleRate(filePath string, sampleRate int) error {
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", "-ar", fmt.Sprintf("%d", sampleRate), filePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error playing sound with ffplay: %v", err)
	}
	return cmd.Wait()
}

// Done uses pgrep to check if ffplay is running
func (player *Player) Done() bool {
	cmd := exec.Command("pgrep", "ffplay")
	err := cmd.Start()
	if err != nil {
		panic(err) // pgrep not found
	}
	err = cmd.Wait()
	if err != nil {
		return true // not found with pgrep
	}
	return false // no errors, pgrep found ffplay running
}
