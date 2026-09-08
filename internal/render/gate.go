package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// Only visible gameplay sprites affect cutaway: hidden enemies/items must not
// leak their presence through holes in scenery. Recomputed outside forest cache
// so picking up loot or moving enemies updates the gate without a player step.
func (r *Renderer) gateForeground(s *domain.Session, vis domain.Visibility, tick int, gate image.Rectangle) []image.Rectangle {
	var protected []image.Rectangle
	add := func(b image.Rectangle) {
		if b.Inset(-6).Overlaps(gate) {
			protected = append(protected, b)
		}
	}
	for _, it := range s.Level.Items {
		if !vis.Visible[domain.Point{X: it.X, Y: it.Y}] {
			continue
		}
		add(image.Rect(it.X*TileSize-1, it.Y*TileSize-1, (it.X+1)*TileSize+1, (it.Y+1)*TileSize+1))
	}
	hero := func(role string, x, y int) {
		img, outlined := r.sprites.heroFrame(role, tick)
		if img == nil {
			return
		}
		pad := 0
		if outlined {
			pad = worldOutlineRadius
		}
		size := img.Bounds().Size()
		left, top := x*TileSize+TileSize/2-size.X/2, (y+1)*TileSize-size.Y+pad
		add(image.Rect(left, top, left+size.X, top+size.Y))
	}
	for _, op := range s.Level.AllOpponents() {
		if op.IsAlive() && op.IsVisible && vis.Visible[domain.Point{X: op.X, Y: op.Y}] {
			hero(op.Type.SpriteRole(), op.X, op.Y)
		}
	}
	hero("player", s.Player.X, s.Player.Y)
	return protected
}

// A soft local cutaway clears visual space around the whole pickup (including
// transparent arch air), avoiding the illusion that loot is inside the gate.
func gateAlpha(p image.Point, protected []image.Rectangle) float32 {
	a := float32(1)
	for _, b := range protected {
		dx := max(b.Min.X-p.X, 0, p.X-b.Max.X)
		dy := max(b.Min.Y-p.Y, 0, p.Y-b.Max.Y)
		t := float32(max(dx, dy)-2) / 4
		t = max(float32(0), min(float32(1), t))
		a = min(a, .12+.88*t*t*(3-2*t))
	}
	return a
}

func (r *Renderer) drawGate(dst *ebiten.Image, s *domain.Session, vis domain.Visibility, camX, camY, tick int) {
	p := s.Level.Exit
	if !vis.Visible[p] && !vis.Explored[p] {
		return
	}
	img := r.sprites.Frame("portal", tick)
	if img == nil {
		return
	}
	b := portalBounds(p)
	if !b.Overlaps(image.Rect(camX, camY, camX+r.Layout.GridW, camY+r.Layout.GridH)) {
		return
	}
	protected := r.gateForeground(s, vis, tick, b)
	light := float32(1)
	if !vis.Visible[p] {
		light = 1 - float32(ExploredDim)/255
	}
	if len(protected) == 0 {
		r.drawForestProp(dst, forestProp{role: "portal", frame: tick, rect: b}, camX, camY, light)
		return
	}
	// One small mesh draw, no per-frame texture allocation or GPU readback.
	const step = 2
	cols, rows := b.Dx()/step+1, b.Dy()/step+1
	vertices := make([]ebiten.Vertex, 0, cols*rows)
	indices := make([]uint16, 0, (cols-1)*(rows-1)*6)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pos := b.Min.Add(image.Pt(x*step, y*step))
			vertices = append(vertices, ebiten.Vertex{DstX: float32(pos.X - camX), DstY: float32(pos.Y - camY),
				SrcX:   float32(img.Bounds().Min.X) + float32(x*step)*float32(img.Bounds().Dx())/float32(b.Dx()),
				SrcY:   float32(img.Bounds().Min.Y) + float32(y*step)*float32(img.Bounds().Dy())/float32(b.Dy()),
				ColorR: light, ColorG: light, ColorB: light, ColorA: gateAlpha(pos, protected)})
			if x < cols-1 && y < rows-1 {
				i := uint16(y*cols + x)
				c := uint16(cols)
				indices = append(indices, i, i+1, i+c, i+1, i+c+1, i+c)
			}
		}
	}
	dst.DrawTriangles(vertices, indices, img, nil)
}
