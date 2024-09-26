//go:build !ff

package playsample

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

var (
	audioDevice *sdl.AudioDeviceID
	mut         sync.Mutex
)

// Player represents a struct with methods for playing audio
type Player struct {
	Initialized bool
}

// NewPlayer tries to initialize SDL2 Audio and returns a Player struct
func NewPlayer() *Player {
	var player Player
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		log.Printf("could not initialize SDL: %v", err)
		return &player
	}
	return &Player{Initialized: true}
}

// Close calls sdl.Quit and sets the Player struct as uninitialized
func (player *Player) Close() {
	sdl.Quit()
	player.Initialized = false
}

// PlayWav plays a WAV file using SDL2 and SDL_mixer
func (player *Player) PlayWav(filePath string) error {
	if !player.Initialized {
		return errors.New("SDL2 Audio needs to be initialized first")
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
	if err := music.Play(1); err != nil {
		return fmt.Errorf("could not play music: %v", err)
	}
	// Wait until the music is finished playing
	for mix.PlayingMusic() {
		sdl.Delay(100)
	}
	return nil
}

func (player *Player) PlayWaveform(samples []float64, sampleRate, bitDepth, channels int) error {
	if !player.Initialized {
		return errors.New("SDL2 Audio needs to be initialized first")
	}
	var desired, obtained sdl.AudioSpec
	desired.Freq = int32(sampleRate)
	desired.Channels = uint8(channels)
	desired.Samples = 4096
	switch bitDepth {
	case 8:
		desired.Format = sdl.AUDIO_S8
	case 16:
		desired.Format = sdl.AUDIO_S16LSB
	case 24:
		desired.Format = sdl.AUDIO_S32LSB
	case 32:
		desired.Format = sdl.AUDIO_F32SYS
	default:
		return fmt.Errorf("unsupported bit depth: %d", bitDepth)
	}
	audioDeviceID, err := sdl.OpenAudioDevice("", false, &desired, &obtained, 0)
	if err != nil {
		return fmt.Errorf("could not open audio device: %v", err)
	}
	mut.Lock()
	audioDevice = &audioDeviceID
	mut.Unlock()
	defer func() {
		mut.Lock()
		if *audioDevice == audioDeviceID {
			audioDevice = nil
		}
		sdl.CloseAudioDevice(audioDeviceID)
		mut.Unlock()
	}()
	if obtained.Format != desired.Format || obtained.Channels != desired.Channels || obtained.Freq != desired.Freq {
		return fmt.Errorf("obtained audio spec does not match desired spec")
	}
	buf := new(bytes.Buffer)
	switch bitDepth {
	case 8:
		for _, sample := range samples {
			sampleInt8 := int8(sample * 127)
			if err := binary.Write(buf, binary.LittleEndian, sampleInt8); err != nil {
				return fmt.Errorf("error converting float64 to int8: %v", err)
			}
		}
	case 16:
		for _, sample := range samples {
			sampleInt16 := int16(sample * 32767)
			if err := binary.Write(buf, binary.LittleEndian, sampleInt16); err != nil {
				return fmt.Errorf("error converting float64 to int16: %v", err)
			}
		}
	case 24:
		for _, sample := range samples {
			sampleInt32 := int32(sample * 8388607)
			bytes24 := make([]byte, 3)
			bytes24[0] = byte(sampleInt32 & 0xFF)
			bytes24[1] = byte((sampleInt32 >> 8) & 0xFF)
			bytes24[2] = byte((sampleInt32 >> 16) & 0xFF)
			if _, err := buf.Write(bytes24); err != nil {
				return fmt.Errorf("error converting float64 to 24-bit: %v", err)
			}
		}
	case 32:
		for _, sample := range samples {
			sampleFloat32 := float32(sample)
			if err := binary.Write(buf, binary.LittleEndian, sampleFloat32); err != nil {
				return fmt.Errorf("error converting float64 to float32: %v", err)
			}
		}
	default:
		return fmt.Errorf("unsupported bit depth: %d", bitDepth)
	}
	if err := sdl.QueueAudio(audioDeviceID, buf.Bytes()); err != nil {
		return fmt.Errorf("could not queue audio data: %v", err)
	}
	sdl.PauseAudioDevice(audioDeviceID, false)
	for sdl.GetQueuedAudioSize(*audioDevice) > 0 {
		sdl.Delay(100)
	}
	return nil
}

func (player *Player) Done() bool {
	return !mix.PlayingMusic()
}
