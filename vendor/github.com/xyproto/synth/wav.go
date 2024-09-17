package synth

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

// SaveToWav saves the waveform to a .wav file using int16 PCM format.
func SaveToWav(w io.WriteSeeker, samples []float64, sampleRate int) error {
	if len(samples) == 0 {
		return fmt.Errorf("cannot save empty waveform: no samples provided")
	}

	// Create a new WAV encoder for int16 PCM
	enc := wav.NewEncoder(w, sampleRate, 16, 1, 1) // 16-bit, mono channel

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

// FFPlayWav plays a WAV file using ffplay
func FFPlayWav(filePath string) error {
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", filePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("error playing sound with ffplay: %v", err)
	}
	return cmd.Wait()
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

// PlayKick generates and plays the current kick drum sound
func (cfg *Settings) PlayKick() error {
	samples, err := cfg.GenerateKickWaveform()
	if err != nil {
		return err
	}
	return PlayWaveform(samples, cfg.SampleRate)
}

// PlayWav plays a WAV file using SDL2 and SDL_mixer
func PlayWav(filePath string) error {
	// Initialize SDL
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		return fmt.Errorf("could not initialize SDL: %v", err)
	}
	defer sdl.Quit()

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
func PlayWaveform(samples []float64, sampleRate int) error {
	// Initialize SDL
	if err := sdl.Init(sdl.INIT_AUDIO); err != nil {
		return fmt.Errorf("could not initialize SDL: %v", err)
	}
	defer sdl.Quit()

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
