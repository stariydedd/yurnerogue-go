// Package render рисует игру средствами Ebitengine: карту, HUD и экраны меню.
// Зависит от domain, но не наоборот.
package render

import (
	"math"

	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

// Размер тайла и видимой области. Исходные спрайты 16px, множитель 2.
const (
	SpriteScale = 2
	TileSize    = 16 * SpriteScale

	ViewCols = 40
	ViewRows = 22
)

// Layout — размеры кадра. Меняется на портретную «консоль» на тач-устройствах,
// поэтому рендер берёт значения отсюда, а не из констант.
type Layout struct {
	Language         locale.Language
	ScreenW, ScreenH int
	GridW, GridH     int // игровое поле
	PanelH           int // статус-панель
	ControlsH        int // панель экранных кнопок, 0 на десктопе
	Touch            bool
}

// DesktopLayout — раскладка с клавиатурой: поле 40x22 тайла и панель снизу.
func DesktopLayout() Layout {
	gridW, gridH := ViewCols*TileSize, ViewRows*TileSize
	panelH := 128
	return Layout{
		ScreenW: gridW,
		ScreenH: gridH + panelH,
		GridW:   gridW,
		GridH:   gridH,
		PanelH:  panelH,
	}
}

// DesktopLayoutForSize expands the viewport instead of distorting a fixed
// frame. Keep the reference HUD/menu size and fit it with one uniform scale.
func DesktopLayoutForSize(width, height int) Layout {
	l := DesktopLayout()
	if width <= 0 || height <= 0 {
		return l
	}
	scale := math.Min(float64(width)/float64(l.ScreenW), float64(height)/float64(l.ScreenH))
	l.ScreenW = max(l.ScreenW, int(math.Floor(float64(width)/scale)))
	l.ScreenH = max(l.ScreenH, int(math.Floor(float64(height)/scale)))
	l.GridW, l.GridH = l.ScreenW, l.ScreenH-l.PanelH
	return l
}

// TouchLayout — портретная раскладка в стиле ретро-консоли: карта сверху,
// статус-панель, снизу экранные кнопки. Высота повторяет пропорции окна,
// чтобы канвас не растягивало.
func TouchLayout(windowW, windowH int) Layout {
	const screenW = 480
	ratio := float64(windowH) / float64(max(1, windowW))
	if ratio < 1.6 {
		ratio = 1.6
	}
	if ratio > 2.3 {
		ratio = 2.3
	}
	screenH := int(float64(screenW) * ratio)

	panelH := 156
	controlsH := 228
	return Layout{
		ScreenW:   screenW,
		ScreenH:   screenH,
		GridW:     screenW,
		GridH:     screenH - panelH - controlsH,
		PanelH:    panelH,
		ControlsH: controlsH,
		Touch:     true,
	}
}

// GridTop — верхняя граница статус-панели (она же низ игрового поля).
func (l Layout) GridTop() int { return l.GridH }

// ControlsTop — верхняя граница панели кнопок.
func (l Layout) ControlsTop() int { return l.GridH + l.PanelH }
