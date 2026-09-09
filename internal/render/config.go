// Package render рисует игру средствами Ebitengine: карту, HUD и экраны меню.
// Зависит от domain, но не наоборот.
package render

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
	ScreenW, ScreenH int
	GridW, GridH     int // игровое поле
	PanelH           int // статус-панель
	ControlsH        int // панель экранных кнопок, 0 на десктопе
	Touch            bool
}

// DesktopLayout — раскладка с клавиатурой: поле 40x22 тайла и панель снизу.
func DesktopLayout() Layout {
	gridW, gridH := ViewCols*TileSize, ViewRows*TileSize
	panelH := 144
	return Layout{
		ScreenW: gridW,
		ScreenH: gridH + panelH,
		GridW:   gridW,
		GridH:   gridH,
		PanelH:  panelH,
	}
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

	panelH := 108
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
