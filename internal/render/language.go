package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func (r *Renderer) tr(key string) string { return locale.Text(r.Layout.Language, key) }

func (r *Renderer) translateMessage(message string) string {
	return locale.Message(r.Layout.Language, message)
}

func helpEntryName(l Layout, entry helpEntry) string {
	if l.Language == locale.Russian {
		if category, ok := map[string]string{
			"food": "FOOD", "elixir": "CLARITY", "scroll": "SCROLL", "sword": "WEAPON", "portal": "PORTAL",
		}[entry.role]; ok {
			return locale.Text(l.Language, category)
		}
	}
	return entry.name
}

// Language targets are pointer-only and intentionally absent from MenuButtons.
func LanguageTargets(l Layout) map[locale.Language]image.Rectangle {
	x, y := l.ScreenW/2-80, l.ScreenH-72
	return map[locale.Language]image.Rectangle{
		locale.English: image.Rect(x, y, x+72, y+48),
		locale.Russian: image.Rect(x+88, y, x+160, y+48),
	}
}

func LanguageAt(l Layout, x, y int) locale.Language {
	for language, box := range LanguageTargets(l) {
		if image.Pt(x, y).In(box) {
			return language
		}
	}
	return ""
}

func (r *Renderer) DrawLanguageControls(dst *ebiten.Image) {
	for language, box := range LanguageTargets(r.Layout) {
		r.uiSlot(dst, box, locale.Normalize(r.Layout.Language) == language)
		flag := image.Rect(box.Min.X+12, box.Min.Y+10, box.Max.X-12, box.Max.Y-10)
		drawFlag(dst, flag, language)
	}
}

func drawFlag(dst *ebiten.Image, box image.Rectangle, language locale.Language) {
	white := color.RGBA{245, 245, 239, 255}
	blue := color.RGBA{22, 54, 137, 255}
	red := color.RGBA{205, 38, 52, 255}
	if language == locale.Russian {
		fillBox(dst, box, white)
		fillBox(dst, image.Rect(box.Min.X, box.Min.Y+box.Dy()/3, box.Max.X, box.Min.Y+2*box.Dy()/3), blue)
		fillBox(dst, image.Rect(box.Min.X, box.Min.Y+2*box.Dy()/3, box.Max.X, box.Max.Y), red)
		return
	}
	fillBox(dst, box, blue)
	// Pixel-stepped diagonals preserve crisp edges at every display scale.
	for x := 0; x < box.Dx(); x++ {
		y := x * (box.Dy() - 1) / (box.Dx() - 1)
		for _, row := range []int{y, box.Dy() - 1 - y} {
			fillBox(dst, image.Rect(box.Min.X+x, max(box.Min.Y, box.Min.Y+row-3), box.Min.X+x+1, min(box.Max.Y, box.Min.Y+row+4)), white)
			fillBox(dst, image.Rect(box.Min.X+x, max(box.Min.Y, box.Min.Y+row-1), box.Min.X+x+1, min(box.Max.Y, box.Min.Y+row+1)), red)
		}
	}
	c := boxCenter(box)
	fillBox(dst, image.Rect(c.X-5, box.Min.Y, c.X+5, box.Max.Y), white)
	fillBox(dst, image.Rect(box.Min.X, c.Y-5, box.Max.X, c.Y+5), white)
	fillBox(dst, image.Rect(c.X-3, box.Min.Y, c.X+3, box.Max.Y), red)
	fillBox(dst, image.Rect(box.Min.X, c.Y-3, box.Max.X, c.Y+3), red)
}
