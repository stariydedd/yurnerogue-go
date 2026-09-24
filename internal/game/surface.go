package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

var _ ebiten.FinalScreenDrawer = (*Game)(nil)

// DrawFinalScreen scales the original logical surface directly to device pixels.
// The regular offscreen uses CSS/window pixels: shrinking 480px to a phone's
// 390px and enlarging that to 1170px loses detail even with nearest filtering.
// Layout remains in window coordinates so pointer mapping is unchanged.
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

// Drawing and pointer input share this exact transform. The desktop can leave
// a subpixel rounding margin, but never stretches sprites along a single axis.
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
