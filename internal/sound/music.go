package sound

import (
	"bytes"
	"encoding/binary"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// musicTrack: одна аранжировка, играющая по кругу. Браузерная сборка играет её
// через Web Audio (music_js.go); в остальных сборках это плеер Ebitengine.
type musicTrack interface {
	Play()
	Pause()
	IsPlaying() bool
	SetVolume(float64)
}

// ebitenTrack играет по кругу 16-битный стерео PCM через микшер Ebitengine.
func ebitenTrack(context *audio.Context, pcm []byte) musicTrack {
	p, err := context.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm))))
	if err != nil {
		return nil
	}
	return p
}

// stereoFloat32 разделяет 16-битный стерео PCM на float32-срез для каждого канала.
func stereoFloat32(pcm []byte) (left, right []float32) {
	frames := len(pcm) / 4
	left, right = make([]float32, frames), make([]float32, frames)
	for i := 0; i < frames; i++ {
		left[i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*4:]))) / (1 << 15)
		right[i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*4+2:]))) / (1 << 15)
	}
	return left, right
}
