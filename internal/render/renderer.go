package render

import (
	"bytes"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/stariydedd/yurnerogue-go/internal/assets"
)

// Fonts — пиксельный Press Start 2P в нужных размерах. Шрифт зашит в бинарник,
// поэтому выглядит одинаково на десктопе и в браузере.
type Fonts struct {
	Title   text.Face // заголовки экранов
	Menu    text.Face // пункты меню
	UI      text.Face // основной текст HUD
	Compact text.Face // длинные строки на узком экране
	Small   text.Face // подсказки
}

func loadFonts() (*Fonts, error) {
	data, err := assets.FS.ReadFile("fonts/PressStart2P.ttf")
	if err != nil {
		return nil, err
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	face := func(size float64) text.Face {
		f := &text.GoTextFace{Source: src, Size: size}
		// Press Start 2P содержит лигатуры fl и fi, и шейпер подставляет их
		// по умолчанию: пара букв становится одним глифом шириной в клетку.
		// Для пиксельного шрифта это бессмысленно, а таблицу рекордов ломает —
		// имя вроде «fle» занимает две клетки вместо трёх и сдвигает колонки.
		for _, feature := range []string{"liga", "clig"} {
			f.SetFeature(text.MustParseTag(feature), 0)
		}
		return f
	}
	return &Fonts{
		Title:   face(40),
		Menu:    face(18),
		UI:      face(14),
		Compact: face(12),
		Small:   face(10),
	}, nil
}

// Renderer рисует все экраны игры.
type Renderer struct {
	Layout  Layout
	Fonts   *Fonts
	sprites *Sprites
	dim     *ebiten.Image // полупрозрачный слой тумана войны
	tick    int
	forest  *forestCache
}

// New создаёт рендерер под заданную раскладку.
func New(l Layout) (*Renderer, error) {
	sprites, err := LoadSprites()
	if err != nil {
		return nil, err
	}
	fonts, err := loadFonts()
	if err != nil {
		return nil, err
	}
	dim := ebiten.NewImage(TileSize, TileSize)
	dim.Fill(color.RGBA{0, 0, 0, ExploredDim})

	return &Renderer{Layout: l, Fonts: fonts, sprites: sprites, dim: dim}, nil
}

// Tick продвигает счётчик кадров: от него зависят idle-анимации.
func (r *Renderer) Tick() { r.tick++ }

// Sprites даёт доступ к спрайтам (нужен панели экранных кнопок).
func (r *Renderer) Sprites() *Sprites { return r.sprites }

// AnimTick — текущий кадр idle-анимаций.
func (r *Renderer) AnimTick() int { return r.tick / AnimFrameTicks }

// Text рисует строку с якорем в левом верхнем углу.
func (r *Renderer) Text(dst *ebiten.Image, s string, face text.Face, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// TextCentered рисует строку по центру экрана по горизонтали.
func (r *Renderer) TextCentered(dst *ebiten.Image, s string, face text.Face, y float64, clr color.Color) {
	w, _ := text.Measure(s, face, 0)
	r.Text(dst, s, face, float64(r.Layout.ScreenW)/2-w/2, y, clr)
}

// TextRight рисует строку, прижатую правым краем к x.
func (r *Renderer) TextRight(dst *ebiten.Image, s string, face text.Face, x, y float64, clr color.Color) {
	w, _ := text.Measure(s, face, 0)
	r.Text(dst, s, face, x-w, y, clr)
}

// TextWidth — ширина строки в пикселях.
func TextWidth(s string, face text.Face) float64 {
	w, _ := text.Measure(s, face, 0)
	return w
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
