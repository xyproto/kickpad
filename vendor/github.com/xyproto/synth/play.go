//go:build !sdl2

package synth

import (
	"errors"
	"fmt"
	"os"

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

// GeneratePlay generates a waveform of the given type and then plays it
func (player *Player) GeneratePlay(t string, cfg *Settings) error {
	if !player.Initialized {
		return errors.New("ffplay culd not be found")
	}
	return FFGeneratePlay(t, cfg)
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
