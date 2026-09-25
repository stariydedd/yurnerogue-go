//go:build !js

package sound

import "github.com/hajimehoshi/ebiten/v2/audio"

func newMusicTrack(context *audio.Context, pcm []byte) musicTrack {
	return ebitenTrack(context, pcm)
}
