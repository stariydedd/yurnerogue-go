package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

func isHeroRole(role string) bool {
	switch role {
	case "player", "pudge", "bloodseeker", "riki", "axe", "skywrath":
		return true
	}
	return false
}

// heroFrame preserves the animation index and leaves UI frames untouched.
func (s *Sprites) heroFrame(role string, tick int) (*ebiten.Image, bool) {
	frames := s.worldHeroes[role]
	if len(frames) == 0 {
		return s.Frame(role, tick), false
	}
	return frames[((tick%len(frames))+len(frames))%len(frames)], true
}

// Mirror the padded frame as a whole. Compensating for bottom padding keeps
// the original feet and horizontal centre anchored even for odd-width frames.
func heroDrawOptions(size image.Point, x, y, camX, camY, facing int, outlined bool) *ebiten.DrawImageOptions {
	op := &ebiten.DrawImageOptions{}
	pad := 0
	if outlined {
		pad = worldOutlineRadius
		op.ColorScale.Scale(worldSpriteBrightness, worldSpriteBrightness, worldSpriteBrightness, 1)
	}
	if facing < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(float64(size.X), 0)
	}
	op.GeoM.Translate(float64(x*TileSize+TileSize/2-size.X/2-camX), float64((y+1)*TileSize-size.Y+pad-camY))
	return op
}
