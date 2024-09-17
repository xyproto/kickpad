package synth

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"

	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

type SDL2 struct {
	Initialized bool
}

func NewSDL2() *SDL2 {
	var sdl2 SDL2
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		log.Printf("could not initialize SDL: %v", err)
		sdl2.Initialized = false
	}
	sdl2.Initialized = true
	return &sdl2
}
func (sdl2 *SDL2) Close() {
	sdl.Quit()
}

// PlayKick generates and plays the current kick drum sound
func (sdl2 *SDL2) PlayKick(cfg *Settings) error {
	if !sdl2.Initialized {
		return errors.New("SDL2 audio needs to be initialized first")
	}

	samples, err := cfg.GenerateKickWaveform()
	if err != nil {
		return err
	}
	return sdl2.PlayWaveform(samples, cfg.SampleRate)
}

// PlayWav plays a WAV file using SDL2 and SDL_mixer
func (sdl2 *SDL2) PlayWav(filePath string) error {
	if !sdl2.Initialized {
		return errors.New("SDL2 audio needs to be initialized first")
	}

	// Open the audio device
	if err := mix.OpenAudio(44100, sdl.AUDIO_S16SYS, 2, 1024); err != nil {
		return fmt.Errorf("could not open audio: %v", err)
	}
	defer mix.CloseAudio()

	// Load the WAV file
	music, err := mix.LoadMUS(filePath)
	if err != nil {
		return fmt.Errorf("could not load music file: %v", err)
	}
	defer music.Free()

	// Play the music
	if err := music.Play(1); err != nil {
		return fmt.Errorf("could not play music: %v", err)
	}

	// Wait until the music is finished playing
	for mix.PlayingMusic() {
		sdl.Delay(100)
	}

	return nil
}

// PlayWaveform plays raw waveform samples using SDL2
func (sdl2 *SDL2) PlayWaveform(samples []float64, sampleRate int) error {
	var desired, obtained sdl.AudioSpec

	desired.Freq = int32(sampleRate)
	desired.Format = sdl.AUDIO_F32SYS // 32-bit float samples
	desired.Channels = 1              // Mono
	desired.Samples = 4096            // Buffer size

	// Open the audio device
	audioDeviceID, err := sdl.OpenAudioDevice("", false, &desired, &obtained, 0)
	if err != nil {
		return fmt.Errorf("could not open audio device: %v", err)
	}
	defer sdl.CloseAudioDevice(audioDeviceID)

	// Check if obtained spec matches desired spec
	if obtained.Format != desired.Format || obtained.Channels != desired.Channels || obtained.Freq != desired.Freq {
		return fmt.Errorf("obtained audio spec does not match desired spec")
	}

	// Convert samples []float64 to []float32
	audioData := make([]float32, len(samples))
	for i, sample := range samples {
		audioData[i] = float32(sample)
	}

	// Convert audioData []float32 to []byte
	buf := new(bytes.Buffer)
	for _, f := range audioData {
		if err := binary.Write(buf, binary.LittleEndian, f); err != nil {
			return fmt.Errorf("error converting float32 to bytes: %v", err)
		}
	}

	// Queue the audio data
	if err := sdl.QueueAudio(audioDeviceID, buf.Bytes()); err != nil {
		return fmt.Errorf("could not queue audio data: %v", err)
	}

	// Start playback
	sdl.PauseAudioDevice(audioDeviceID, false)

	// Wait until the audio finishes playing
	for sdl.GetQueuedAudioSize(audioDeviceID) > 0 {
		sdl.Delay(100)
	}

	return nil
}
