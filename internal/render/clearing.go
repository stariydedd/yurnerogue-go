package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

type clearingEntrance struct {
	start  image.Point
	dir    domain.Point
	length int
}

type clearing struct {
	bounds    image.Rectangle
	cells     map[domain.Point]bool
	entrances []clearingEntrance
}

// Build composition from revealed ground only, not Level.Rooms. A decorative
// clearing is a connected patch of room floor, excluding known corridor cells.
func knownClearings(v forestView, paths map[domain.Point]bool) []clearing {
	seen := map[domain.Point]bool{}
	var result []clearing
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if seen[p] || !v.ground[p] || paths[p] {
				continue
			}
			c := clearing{cells: map[domain.Point]bool{p: true}}
			queue := []domain.Point{p}
			seen[p] = true
			for i := 0; i < len(queue); i++ {
				p := queue[i]
				c.bounds = c.bounds.Union(image.Rect(p.X*32, p.Y*32, (p.X+1)*32, (p.Y+1)*32))
				for _, d := range forestDirs {
					q := domain.Point{X: p.X + d.X, Y: p.Y + d.Y}
					if !v.ground[q] {
						continue
					}
					if paths[q] {
						c.entrances = append(c.entrances, clearingEntrance{
							start: image.Pt(p.X*32+16+d.X*16, p.Y*32+16+d.Y*16),
							dir:   domain.Point{X: -d.X, Y: -d.Y}, length: 64 + cellHash(p.X, p.Y)%33})
					} else if !seen[q] {
						seen[q] = true
						c.cells[q] = true
						queue = append(queue, q)
					}
				}
			}
			if c.bounds.Dx() >= 96 && c.bounds.Dy() >= 96 {
				result = append(result, c)
			}
		}
	}
	return result
}

// Positive distance is quiet meadow; negative distance is walkable low moss
// and foliage. The composition changes at ROOM scale, not every 32px tile.
func (c clearing) distance(x, y int) float64 {
	b := c.bounds
	scale := math.Min(1, float64(min(b.Dx(), b.Dy()))/256)
	fx, fy := float64(x), float64(y)
	inset := 32 * scale
	radius := 92 * scale
	cx, cy := float64(b.Min.X+b.Max.X)/2, float64(b.Min.Y+b.Max.Y)/2
	hx, hy := float64(b.Dx())/2-inset, float64(b.Dy())/2-inset
	dx, dy := math.Abs(fx-cx)-hx+radius, math.Abs(fy-cy)-hy+radius
	d := radius - math.Hypot(math.Max(dx, 0), math.Max(dy, 0)) - math.Min(math.Max(dx, dy), 0)
	// Broad asymmetric lobes, with a finer broken leaf edge.
	wave := math.Sin(fx/57+float64(b.Min.Y))*18 + math.Sin(fy/43+float64(b.Min.X))*14 + math.Sin((fx+fy)/29)*6
	d += wave * scale
	for _, e := range c.entrances {
		along, across := e.coordinates(x, y)
		if along >= 0 && along < float64(e.length+16) {
			d = math.Max(d, 22-math.Abs(across))
		}
	}
	return d
}

func (e clearingEntrance) coordinates(x, y int) (float64, float64) {
	dx, dy := x-e.start.X, y-e.start.Y
	return float64(dx*e.dir.X + dy*e.dir.Y), float64(dx*e.dir.Y - dy*e.dir.X)
}

func (c clearing) contains(rect image.Rectangle) bool {
	if !rect.In(c.bounds) {
		return false
	}
	for y := rect.Min.Y / 32; y <= (rect.Max.Y-1)/32; y++ {
		for x := rect.Min.X / 32; x <= (rect.Max.X-1)/32; x++ {
			if !c.cells[domain.Point{X: x, Y: y}] {
				return false
			}
		}
	}
	return true
}

func (c clearing) approach(rect image.Rectangle) bool {
	for _, e := range c.entrances {
		along, across := e.coordinates((rect.Min.X+rect.Max.X)/2, (rect.Min.Y+rect.Max.Y)/2)
		if along > -24 && along < float64(e.length+20) && math.Abs(across) < float64(20+max(rect.Dx(), rect.Dy())/2) {
			return true
		}
	}
	return false
}

func (c clearing) plantFits(v forestView, rect image.Rectangle) bool {
	for y := max(0, rect.Min.Y/32); y <= min(domain.Rows-1, (rect.Max.Y-1)/32); y++ {
		for x := max(0, rect.Min.X/32); x <= min(domain.Cols-1, (rect.Max.X-1)/32); x++ {
			p := domain.Point{X: x, Y: y}
			if v.ground[p] && !c.cells[p] && rect.Overlaps(image.Rect(x*32+5, y*32+5, x*32+27, y*32+27)) {
				return false
			}
		}
	}
	return true
}

func (c clearing) plants(v forestView) []forestProp {
	var props []forestProp
	for gy := (c.bounds.Min.Y - 16) / 12; gy <= (c.bounds.Max.Y+16)/12; gy++ {
		for gx := (c.bounds.Min.X - 16) / 12; gx <= (c.bounds.Max.X+16)/12; gx++ {
			h := cellHash(gx+173, gy+391)
			x, y := gx*12+h%11-5, gy*12+(h/13)%11-5
			d := c.distance(x, y)
			if d > 10 || (d > 0 && h%3 != 0) {
				continue
			}
			w, height := 28+h%17, 18+(h/37)%11
			role, frame := "bush", (h/71)%3
			if h%5 == 0 {
				role, frame = "verge", h%4
				w, height = 36+h%17, 16
			}
			rect := image.Rect(x-w/2, y-height/2, x+w/2, y+height/2)
			if !c.plantFits(v, rect) || c.approach(rect) {
				continue
			}
			props = append(props, forestProp{role: role, frame: frame, rect: rect, groundCover: true})
		}
	}
	return props
}

func (r *Renderer) drawClearingBanks(dst *ebiten.Image, v forestView, vis domain.Visibility, paths map[domain.Point]bool, camX, camY int) {
	viewport := dst.Bounds().Add(image.Pt(camX, camY))
	for _, c := range knownClearings(v, paths) {
		if !c.bounds.Overlaps(viewport) {
			continue
		}
		// A textured, low moss bed defines the organic clearing silhouette.
		// It is NOT a wall: all original floor remains opaque and walkable.
		area := c.bounds.Intersect(viewport)
		for y := area.Min.Y / 4 * 4; y < area.Max.Y; y += 4 {
			for x := area.Min.X / 4 * 4; x < area.Max.X; x += 4 {
				rect := image.Rect(x, y, x+4, y+4)
				if !c.contains(rect) {
					continue
				}
				d := c.distance(x+2, y+2)
				if d > 16 {
					continue
				}
				alpha := float32(math.Min(.46, math.Max(0, (16-d)/100)))
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(.125, .125)
				op.GeoM.Translate(float64(x-camX), float64(y-camY))
				op.ColorScale.ScaleAlpha(alpha)
				dst.DrawImage(r.dim, op)
			}
		}
		// A known entrance tapers into the meadow, instead of stopping at a
		// square door tile. Unknown passages cannot create these approach marks.
		for _, e := range c.entrances {
			for along := 0; along < e.length; along += 2 {
				width := int(14 * math.Pow(1-float64(along)/float64(e.length), .65))
				width += cellHash(e.start.X+along/4, e.start.Y)%3 - 1
				for across := -width; across < width; across += 2 {
					x := e.start.X + along*e.dir.X + across*e.dir.Y
					y := e.start.Y + along*e.dir.Y - across*e.dir.X
					rect := image.Rect(x, y, x+2, y+2)
					if !c.contains(rect) {
						continue
					}
					fade := float32(1 - math.Pow(float64(along)/float64(e.length), 1.4))
					// Fade both texture and its remembered-state dim together.
					// Blending an undimmed trail over dark grass would make it glow.
					r.drawEntranceSample(dst, rect, camX, camY, fade, vis.Visible[domain.Point{X: x / 32, Y: y / 32}])
				}
			}
		}
		for _, p := range c.plants(v) {
			if !p.rect.Overlaps(viewport) {
				continue
			}
			r.drawForestProp(dst, p, camX, camY, clearingPlantLight(v, p.rect))
		}
	}
}

// Low banks are part of the forest, including the portion over walkable room
// floor. A cell-visible boolean would imprint the rectangular room boundary.
func clearingPlantLight(v forestView, rect image.Rectangle) float32 {
	return .86 * v.light(rect)
}

func (r *Renderer) drawEntranceSample(dst *ebiten.Image, rect image.Rectangle, camX, camY int, alpha float32, visible bool) {
	clip := rect.Sub(image.Pt(camX, camY)).Intersect(dst.Bounds())
	if clip.Empty() {
		return
	}
	for y := rect.Min.Y / 32; y <= (rect.Max.Y-1)/32; y++ {
		for x := rect.Min.X / 32; x <= (rect.Max.X-1)/32; x++ {
			frame, sample := groundSample(x, y)
			img := r.sprites.Frame("trail", frame)
			if img == nil {
				continue
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x*32-camX), float64(y*32-camY))
			light := float32(1)
			if !visible {
				light -= float32(ExploredDim) / 255
			}
			op.ColorScale.Scale(light, light, light, 1)
			op.ColorScale.ScaleAlpha(alpha)
			dst.SubImage(clip).(*ebiten.Image).DrawImage(img.SubImage(sample.Add(img.Bounds().Min)).(*ebiten.Image), op)
		}
	}
}
