package sound

import (
	"bytes"
	"encoding/binary"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// musicTrack is one looping arrangement. The browser build plays it through
// Web Audio (music_js.go); elsewhere it is an Ebitengine player.
type musicTrack interface {
	Play()
	Pause()
	IsPlaying() bool
	SetVolume(float64)
}

// ebitenTrack loops 16-bit stereo PCM through the Ebitengine mixer.
func ebitenTrack(context *audio.Context, pcm []byte) musicTrack {
	p, err := context.NewPlayer(audio.NewInfiniteLoop(bytes.NewReader(pcm), int64(len(pcm))))
	if err != nil {
		return nil
	}
	return p
}

// stereoFloat32 splits 16-bit stereo PCM into one float32 slice per channel.
func stereoFloat32(pcm []byte) (left, right []float32) {
	frames := len(pcm) / 4
	left, right = make([]float32, frames), make([]float32, frames)
	for i := 0; i < frames; i++ {
		left[i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*4:]))) / (1 << 15)
		right[i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*4+2:]))) / (1 << 15)
	}
	return left, right
}
