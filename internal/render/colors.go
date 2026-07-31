package render

import "image/color"

// Палитра интерфейса: чёрный фон, золото заголовков, приглушённые подсказки.
var (
	Black     = color.RGBA{0, 0, 0, 255}
	White     = color.RGBA{225, 225, 225, 255}
	Gold      = color.RGBA{212, 175, 55, 255}
	Hilite    = color.RGBA{255, 215, 0, 255}
	MsgColor  = color.RGBA{255, 200, 120, 255}
	HintColor = color.RGBA{140, 140, 140, 255}

	// HP-бар.
	HPBack   = color.RGBA{60, 20, 20, 255}
	HPFill   = color.RGBA{205, 52, 48, 255}
	HPBorder = color.RGBA{18, 6, 6, 255}

	// Диалоги и оверлеи.
	shadowGold = color.RGBA{60, 40, 0, 255}
	dialogBG   = color.RGBA{25, 25, 25, 255}
	menuBG     = color.RGBA{10, 10, 10, 255}
	menuBorder = color.RGBA{90, 90, 90, 255}
	overlayDim = color.RGBA{0, 0, 0, 160}

	// Панель экранных кнопок.
	PanelBG    = color.RGBA{18, 18, 20, 255}
	BtnBase    = color.RGBA{54, 54, 60, 255}
	BtnPressed = color.RGBA{104, 104, 116, 255}
	BtnEdge    = color.RGBA{10, 10, 12, 255}
	ArrowColor = color.RGBA{150, 150, 160, 255}
)

// ExploredDim — затемнение клеток, которые игрок помнит, но не видит сейчас.
const ExploredDim = 175
