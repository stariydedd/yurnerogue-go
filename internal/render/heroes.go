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

// heroFrame сохраняет индекс анимации и не трогает кадры интерфейса.
func (s *Sprites) heroFrame(role string, tick int) (*ebiten.Image, bool) {
	frames := s.worldHeroes[role]
	if len(frames) == 0 {
		return s.Frame(role, tick), false
	}
	return frames[((tick%len(frames))+len(frames))%len(frames)], true
}

// Зеркалим кадр с полями целиком. Поправка на нижнее поле держит
// исходные ноги и горизонтальный центр на месте даже у кадров нечётной ширины.
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
