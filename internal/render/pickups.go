package render

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/hajimehoshi/ebiten/v2"
)

const worldOutlineRadius = 1
const worldSpriteBrightness = 1.12

func isPickupRole(role string) bool {
	switch role {
	case "food", "elixir", "scroll", "sword":
		return true
	}
	return false
}

// outlineSprite builds a world-only silhouette once during asset loading.
// Ignore faint extraction glow, and expand actual edges (including diagonals)
// rather than drawing a rectangle around the tile. Keep the original interior.
func outlineSprite(src image.Image, bounds image.Rectangle) *image.NRGBA {
	const pad = worldOutlineRadius
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx()+2*pad, bounds.Dy()+2*pad))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a < 96*257 {
				continue
			}
			for dy := -pad; dy <= pad; dy++ {
				for dx := -pad; dx <= pad; dx++ {
					p := image.Pt(x+dx, y+dy)
					if p.In(bounds) {
						_, _, _, neighbour := src.At(p.X, p.Y).RGBA()
						if neighbour >= 96*257 {
							continue
						}
					}
					dst.SetNRGBA(p.X-bounds.Min.X+pad, p.Y-bounds.Min.Y+pad, color.NRGBA{A: 255})
				}
			}
		}
	}
	draw.Draw(dst, image.Rect(pad, pad, pad+bounds.Dx(), pad+bounds.Dy()), src, bounds.Min, draw.Over)
	return dst
}

func (r *Renderer) drawPickup(dst *ebiten.Image, role string, x, y, camX, camY, tick int) {
	frames := r.sprites.groundPickups[role]
	if len(frames) == 0 {
		r.drawTile(dst, role, x, y, camX, camY, tick)
		return
	}
	img := frames[((tick%len(frames))+len(frames))%len(frames)]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x*TileSize-camX-worldOutlineRadius), float64(y*TileSize-camY-worldOutlineRadius))
	// A modest boost separates the pickup from similarly coloured foliage.
	// Black outline and source alpha remain unchanged.
	op.ColorScale.Scale(worldSpriteBrightness, worldSpriteBrightness, worldSpriteBrightness, 1)
	dst.DrawImage(img, op)
}
