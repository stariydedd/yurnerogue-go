package render

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	uiInk       = color.RGBA{8, 8, 8, 255}
	uiEdge      = color.RGBA{187, 79, 25, 255}
	uiHighlight = color.RGBA{255, 183, 110, 255}
	uiText      = color.RGBA{246, 245, 240, 255}
	uiMuted     = color.RGBA{166, 163, 158, 255}
	uiAccent    = color.RGBA{255, 112, 24, 255}
	uiRecess    = color.RGBA{18, 17, 16, 255}
	uiPressed   = color.RGBA{65, 33, 17, 255}
)

// The small stone tile is generated once; panels reuse it without per-frame uploads.
func (r *Renderer) stonePanel(dst *ebiten.Image, box image.Rectangle) {
	if r.uiStone == nil {
		tile := image.NewRGBA(image.Rect(0, 0, 96, 96))
		for y := 0; y < 96; y++ {
			for x := 0; x < 96; x++ {
				h := uint32(x/2+1)*374761393 + uint32(y/2+1)*668265263
				h = (h ^ (h >> 13)) * 1274126177
				n := uint8(h % 4)
				// Subtle diagonal stone seams, periodic across tile boundaries.
				if (x+y)%32 == 0 {
					n += 5
				}
				tile.SetRGBA(x, y, color.RGBA{16 + n, 15 + n, 14 + n, 255})
			}
		}
		r.uiStone = ebiten.NewImageFromImage(tile)
	}
	for y := box.Min.Y; y < box.Max.Y; y += 96 {
		for x := box.Min.X; x < box.Max.X; x += 96 {
			part := image.Rect(0, 0, min(96, box.Max.X-x), min(96, box.Max.Y-y))
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			dst.DrawImage(r.uiStone.SubImage(part).(*ebiten.Image), op)
		}
	}
	strokeBox(dst, box.Inset(1), 2, uiInk)
	strokeChamfer(dst, box.Inset(3), 5, 2, uiEdge)
	strokeChamfer(dst, box.Inset(6), 3, 1, uiAccent)
	vector.StrokeLine(dst, float32(box.Min.X+11), float32(box.Min.Y+4), float32(box.Max.X-11), float32(box.Min.Y+4), 1, uiHighlight, false)
	for _, p := range []image.Point{box.Min.Add(image.Pt(7, 7)), {X: box.Max.X - 8, Y: box.Min.Y + 7}, {X: box.Min.X + 7, Y: box.Max.Y - 8}, box.Max.Sub(image.Pt(8, 8))} {
		fillBox(dst, image.Rect(p.X-2, p.Y-2, p.X+3, p.Y+3), uiInk)
		fillBox(dst, image.Rect(p.X-1, p.Y-1, p.X+2, p.Y+2), uiAccent)
		fillBox(dst, image.Rect(p.X-1, p.Y-1, p.X+1, p.Y), uiText)
	}
}

func strokeChamfer(dst *ebiten.Image, box image.Rectangle, cut int, width float32, clr color.Color) {
	cut = min(cut, box.Dx()/2, box.Dy()/2)
	x, y, right, bottom := box.Min.X, box.Min.Y, box.Max.X, box.Max.Y
	points := [...]image.Point{{x + cut, y}, {right - cut, y}, {right, y + cut}, {right, bottom - cut}, {right - cut, bottom}, {x + cut, bottom}, {x, bottom - cut}, {x, y + cut}}
	for i, p := range points {
		next := points[(i+1)%len(points)]
		vector.StrokeLine(dst, float32(p.X), float32(p.Y), float32(next.X), float32(next.Y), width, clr, false)
	}
}

func (r *Renderer) uiSlot(dst *ebiten.Image, box image.Rectangle, active bool) {
	fillBox(dst, box, uiInk)
	inner := uiRecess
	edge := uiEdge
	if active {
		inner = uiPressed
		edge = uiAccent
	}
	fillBox(dst, box.Inset(4), inner)
	strokeChamfer(dst, box.Inset(2), 3, 2, edge)
	strokeChamfer(dst, box.Inset(4), 2, 1, uiMuted)
	vector.StrokeLine(dst, float32(box.Min.X+6), float32(box.Min.Y+3), float32(box.Max.X-6), float32(box.Min.Y+3), 1, uiHighlight, false)
	if active {
		fillBox(dst, image.Rect(box.Min.X+6, box.Max.Y-7, box.Max.X-6, box.Max.Y-4), uiAccent)
	}
	for _, x := range []int{box.Min.X + 5, box.Max.X - 7} {
		for _, y := range []int{box.Min.Y + 5, box.Max.Y - 7} {
			fillBox(dst, image.Rect(x, y, x+2, y+2), uiHighlight)
		}
	}
}

func fillBox(dst *ebiten.Image, box image.Rectangle, clr color.Color) {
	vector.DrawFilledRect(dst, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), clr, false)
}

func strokeBox(dst *ebiten.Image, box image.Rectangle, width float32, clr color.Color) {
	vector.StrokeRect(dst, float32(box.Min.X), float32(box.Min.Y), float32(box.Dx()), float32(box.Dy()), width, clr, false)
}

func boxCenter(box image.Rectangle) image.Point { return box.Min.Add(image.Pt(box.Dx()/2, box.Dy()/2)) }

// fitLabel truncates at rune boundaries, keeping every HUD field inside its own area.
func fitLabel(s string, face text.Face, width float64) string {
	if TextWidth(s, face) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimSpace(string(runes)) + "..."
		if TextWidth(candidate, face) <= width {
			return candidate
		}
	}
	return ""
}

func (r *Renderer) slotLabel(dst *ebiten.Image, label string, box image.Rectangle, clr color.Color) {
	label = fitLabel(label, r.Fonts.Small, float64(box.Dx()-8))
	p := boxCenter(box)
	x, y := float64(p.X)-TextWidth(label, r.Fonts.Small)/2, float64(p.Y)-5
	r.Text(dst, label, r.Fonts.Small, x+1, y+1, uiInk)
	r.Text(dst, label, r.Fonts.Small, x, y, clr)
}
