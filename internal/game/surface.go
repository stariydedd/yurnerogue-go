package game

import "github.com/hajimehoshi/ebiten/v2"

// Игра рисуется в поверхность фиксированного логического размера, а та
// растягивается на всё окно.
//
// Если вернуть логический размер прямо из Layout, Ebitengine впишет картинку
// в канвас с сохранением пропорций и оставит чёрные поля по краям на любом
// экране, чьи пропорции отличаются от игровых. Растяжение убирает поля ценой
// лёгкого искажения — тот же компромисс, что был в Python-версии.

// ensureSurface создаёт логическую поверхность под текущую раскладку.
func (g *Game) ensureSurface() *ebiten.Image {
	l := g.renderer.Layout
	if g.surface == nil ||
		g.surface.Bounds().Dx() != l.ScreenW || g.surface.Bounds().Dy() != l.ScreenH {
		g.surface = ebiten.NewImage(l.ScreenW, l.ScreenH)
	}
	return g.surface
}

// scaleToWindow — во сколько раз логическая поверхность растянута до окна.
func (g *Game) scaleToWindow() (float64, float64) {
	l := g.renderer.Layout
	if g.outW <= 0 || g.outH <= 0 {
		return 1, 1
	}
	return float64(g.outW) / float64(l.ScreenW), float64(g.outH) / float64(l.ScreenH)
}

// toLogical переводит координаты окна (касания, курсор) в координаты
// логической поверхности, в которых заданы кнопки.
func (g *Game) toLogical(x, y int) (int, int) {
	sx, sy := g.scaleToWindow()
	if sx == 0 || sy == 0 {
		return x, y
	}
	return int(float64(x) / sx), int(float64(y) / sy)
}
