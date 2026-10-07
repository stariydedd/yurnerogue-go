package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

var _ ebiten.FinalScreenDrawer = (*Game)(nil)

// DrawFinalScreen масштабирует исходную логическую поверхность прямо в пиксели устройства.
// Обычный offscreen работает в CSS-пикселях окна: сжатие 480px до 390px телефона
// и растяжение обратно до 1170px теряют детали даже с ближайшим соседом.
// Layout остаётся в координатах окна, поэтому соответствие указателя не меняется.
func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	if g.surface == nil {
		ebiten.DefaultDrawFinalScreen(screen, offscreen, geoM)
		return
	}

	bounds := screen.Bounds()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	sx, sy, x, y := g.surfaceTransform(bounds.Dx(), bounds.Dy())
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(bounds.Min.X)+x, float64(bounds.Min.Y)+y)
	screen.DrawImage(g.surface, op)
}

// ensureSurface создаёт логическую поверхность под текущую раскладку.
func (g *Game) ensureSurface() *ebiten.Image {
	l := g.renderer.Layout
	if g.surface == nil ||
		g.surface.Bounds().Dx() != l.ScreenW || g.surface.Bounds().Dy() != l.ScreenH {
		if g.surface != nil {
			g.surface.Deallocate()
		}
		g.surface = ebiten.NewImage(l.ScreenW, l.ScreenH)
	}
	return g.surface
}

// Отрисовка и ввод указателем используют ровно это преобразование. На десктопе может
// остаться поле от субпиксельного округления, но спрайты никогда не растягиваются по одной оси.
func (g *Game) surfaceTransform(width, height int) (sx, sy, x, y float64) {
	l := g.renderer.Layout
	if width <= 0 || height <= 0 {
		return 1, 1, 0, 0
	}
	sx, sy = float64(width)/float64(l.ScreenW), float64(height)/float64(l.ScreenH)
	if !l.Touch {
		sx = math.Min(sx, sy)
		sy = sx
		x, y = (float64(width)-float64(l.ScreenW)*sx)/2, (float64(height)-float64(l.ScreenH)*sy)/2
	}
	return
}

// toLogical переводит координаты окна (касания, курсор) в координаты
// логической поверхности, в которых заданы кнопки.
func (g *Game) toLogical(x, y int) (int, int) {
	sx, sy, dx, dy := g.surfaceTransform(g.outW, g.outH)
	if sx == 0 || sy == 0 {
		return x, y
	}
	return int(math.Floor((float64(x) - dx) / sx)), int(math.Floor((float64(y) - dy) / sy))
}
