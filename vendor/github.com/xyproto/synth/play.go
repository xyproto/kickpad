package synth

import (
	"errors"

	"github.com/xyproto/playsample"
)

var player = playsample.NewPlayer()

type CloseFunc func()

// GeneratePlay generates and plays the current kick drum sound
func GeneratePlay(t string, cfg *Settings) (error, CloseFunc) {
	if !player.Initialized {
		return errors.New("A Player needs to be initialized first"), func() {}
	}
	samples, err := cfg.Generate(t)
	if err != nil {
		return err, func() {}
	}
	return player.PlayWaveform(samples, cfg.SampleRate, cfg.BitDepth, cfg.Channels), player.Close
}
