package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Keep the full animation cell at an integer scale and position. Using opaque
// bounds per frame would turn the character's idle motion into portrait jitter.
func portraitPlacement(frame image.Point, slot image.Rectangle) (int, image.Point) {
	inner := slot.Inset(6)
	if frame.X <= 0 || frame.Y <= 0 || inner.Empty() {
		return 0, image.Point{}
	}
	scale := min(inner.Dx()/frame.X, inner.Dy()/frame.Y)
	if scale == 0 {
		return 0, image.Point{}
	}
	position := inner.Min.Add(image.Pt((inner.Dx()-frame.X*scale)/2, (inner.Dy()-frame.Y*scale)/2))
	return scale, position
}

func (r *Renderer) drawHUDPortrait(dst *ebiten.Image, slot image.Rectangle) {
	frame := r.sprites.Frame("player", r.AnimTick())
	if frame == nil {
		return
	}
	scale, position := portraitPlacement(frame.Bounds().Size(), slot)
	if scale == 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(position.X), float64(position.Y))
	op.Filter = ebiten.FilterNearest
	dst.DrawImage(frame, op)
}
