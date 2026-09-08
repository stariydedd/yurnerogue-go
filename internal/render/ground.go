package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

type groundSpan struct {
	rect  image.Rectangle
	shade int
}

// Shared world-space waves avoid a fresh notch at every tile boundary.
// The floor may extend slightly into the forest, but never hides the central
// 22x22px of a walkable cell. No collision or hidden map data is consulted.
func groundWave(along, boundary, side int) float64 {
	a := int(math.Floor(float64(along) / 48))
	t := float64(along-a*48) / 48
	t = t * t * (3 - 2*t)
	lo := float64(cellHash(a+side*73, boundary)%13 - 3)
	hi := float64(cellHash(a+1+side*73, boundary)%13 - 3)
	return lo + (hi-lo)*t
}

func groundContour(v forestView, p domain.Point) []groundSpan {
	var edge [4]bool
	for side, d := range forestDirs {
		edge[side] = !v.ground[domain.Point{X: p.X + d.X, Y: p.Y + d.Y}]
	}
	origin := image.Pt(p.X*TileSize, p.Y*TileSize)
	if edge == [4]bool{} {
		return []groundSpan{{rect: image.Rect(0, 0, TileSize, TileSize).Add(origin)}}
	}
	// Quantised two-pixel contour matches the art, with rounded convex corners.
	state := func(x, y int) int {
		if x < 0 && !edge[1] || x >= TileSize && !edge[0] || y < 0 && !edge[3] || y >= TileSize && !edge[2] {
			return -1
		}
		wx, wy := origin.X+x, origin.Y+y
		owner := domain.Point{X: int(math.Floor(float64(wx) / TileSize)), Y: int(math.Floor(float64(wy) / TileSize))}
		if owner != p && v.ground[owner] {
			return -1
		}
		left, right, top, bottom := 0.0, float64(TileSize), 0.0, float64(TileSize)
		if edge[1] {
			left -= groundWave(wy, origin.X, 1)
		}
		if edge[0] {
			right += groundWave(wy, origin.X+TileSize, 0)
		}
		if edge[3] {
			top -= groundWave(wx, origin.Y, 3)
		}
		if edge[2] {
			bottom += groundWave(wx, origin.Y+TileSize, 2)
		}
		fx, fy := float64(x)+1, float64(y)+1
		dist := 99.0
		if edge[0] {
			dist = math.Min(dist, right-fx)
		}
		if edge[1] {
			dist = math.Min(dist, fx-left)
		}
		if edge[2] {
			dist = math.Min(dist, bottom-fy)
		}
		if edge[3] {
			dist = math.Min(dist, fy-top)
		}
		for _, corner := range [][2]int{{1, 3}, {0, 3}, {1, 2}, {0, 2}} {
			if !edge[corner[0]] || !edge[corner[1]] {
				continue
			}
			cx, cy := left+16, top+16
			if corner[0] == 0 {
				cx = right - 16
			}
			if corner[1] == 2 {
				cy = bottom - 16
			}
			dx, dy := fx-cx, fy-cy
			if corner[0] == 1 {
				dx = -dx
			}
			if corner[1] == 3 {
				dy = -dy
			}
			if dx > 0 && dy > 0 {
				dist = math.Min(dist, 16-math.Hypot(dx, dy))
			}
		}
		// Preserve the same core guarantee as solid forest props, including
		// the entire 2px sample when it touches the protected centre.
		if x < 27 && x+2 > 5 && y < 27 && y+2 > 5 {
			dist = math.Max(dist, 1)
		}
		if dist < 0 {
			return -1
		}
		return max(0, 3-int(dist/2))
	}
	var spans []groundSpan
	for y := -10; y < TileSize+10; y += 2 {
		start, previous := -10, state(-10, y)
		for x := -8; x <= TileSize+10; x += 2 {
			next := -1
			if x < TileSize+10 {
				next = state(x, y)
			}
			if next != previous {
				if previous >= 0 {
					spans = append(spans, groundSpan{image.Rect(start, y, x, y+2).Add(origin), previous})
				}
				start, previous = x, next
			}
		}
	}
	return spans
}

func (r *Renderer) drawGroundRect(dst *ebiten.Image, role string, rect image.Rectangle, camX, camY int) {
	clip := rect.Sub(image.Pt(camX, camY)).Intersect(dst.Bounds())
	if clip.Empty() {
		return
	}
	for y := int(math.Floor(float64(rect.Min.Y) / TileSize)); y*TileSize < rect.Max.Y; y++ {
		for x := int(math.Floor(float64(rect.Min.X) / TileSize)); x*TileSize < rect.Max.X; x++ {
			r.drawGroundSample(dst.SubImage(clip).(*ebiten.Image), role, domain.Point{X: x, Y: y}, camX, camY)
		}
	}
}

// A 256px continuous surface is stored as four 128px quadrants. Sampling,
// rather than shrinking a complete picture into each tile, preserves details
// and continuity across cells, turns, viewport edges and camera movement.
func groundSample(x, y int) (int, image.Rectangle) {
	x, y = (x%8+8)%8, (y%8+8)%8
	frame := (y/4)*2 + x/4
	x, y = (x%4)*TileSize, (y%4)*TileSize
	return frame, image.Rect(x, y, x+TileSize, y+TileSize)
}

func (r *Renderer) drawGroundSample(dst *ebiten.Image, role string, p domain.Point, camX, camY int) {
	frame, sample := groundSample(p.X, p.Y)
	img := r.sprites.Frame(role, frame)
	if img == nil {
		return
	}
	sample = sample.Add(img.Bounds().Min)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.X*TileSize-camX), float64(p.Y*TileSize-camY))
	dst.DrawImage(img.SubImage(sample).(*ebiten.Image), op)
}

// Narrow, irregular grass strips soften exposed path edges while retaining
// at least the central 22x22 pixels of every corridor cell as visible trail.
func trailFringe(p domain.Point, side int) []image.Rectangle {
	var strips []image.Rectangle
	for offset := 0; offset < TileSize; offset += 4 {
		along, boundary := p.X*TileSize+offset, p.Y*TileSize
		if side < 2 {
			along, boundary = p.Y*TileSize+offset, p.X*TileSize
		}
		if side == 0 || side == 2 {
			boundary += TileSize
		}
		width := 2 + int((groundWave(along, boundary, side)+3)/4)
		var rect image.Rectangle
		switch side {
		case 0:
			rect = image.Rect(TileSize-width, offset, TileSize, offset+4)
		case 1:
			rect = image.Rect(0, offset, width, offset+4)
		case 2:
			rect = image.Rect(offset, TileSize-width, offset+4, TileSize)
		case 3:
			rect = image.Rect(offset, 0, offset+4, width)
		}
		strips = append(strips, rect.Add(image.Pt(p.X*TileSize, p.Y*TileSize)))
	}
	return strips
}

func trailFringes(v forestView, paths map[domain.Point]bool, p domain.Point) []image.Rectangle {
	var strips []image.Rectangle
	for side, d := range forestDirs {
		q := domain.Point{X: p.X + d.X, Y: p.Y + d.Y}
		// Only revealed neighbours can affect edge treatment; hidden passage
		// geometry must not change how the visible end of a trail looks.
		if v.ground[q] && paths[q] {
			continue
		}
		strips = append(strips, trailFringe(p, side)...)
	}
	return strips
}

// r.dim already contains ExploredDim alpha. Compose the contact shadow with
// that existing opacity, rather than multiplying the fog alpha a second time.
func groundDimScale(shade int, visible bool) float32 {
	shadowScale := float32(shade) * .075
	if visible {
		return shadowScale
	}
	return 1 + (1-float32(ExploredDim)/255)*shadowScale
}

func (r *Renderer) drawWalkableGround(dst *ebiten.Image, v forestView, paths map[domain.Point]bool, p domain.Point, camX, camY int, visible bool) {
	contour := groundContour(v, p)
	cell := image.Rect(p.X*TileSize, p.Y*TileSize, (p.X+1)*TileSize, (p.Y+1)*TileSize)
	fringes := trailFringes(v, paths, p)
	for _, span := range contour {
		r.drawGroundRect(dst, "meadow", span.rect, camX, camY)
		if paths[p] {
			r.drawGroundRect(dst, "trail", span.rect.Intersect(cell), camX, camY)
			for _, fringe := range fringes {
				r.drawGroundRect(dst, "meadow", span.rect.Intersect(fringe), camX, camY)
			}
		}
		alpha := groundDimScale(span.shade, visible)
		if alpha > 0 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(span.rect.Dx())/32, float64(span.rect.Dy())/32)
			op.GeoM.Translate(float64(span.rect.Min.X-camX), float64(span.rect.Min.Y-camY))
			op.ColorScale.ScaleAlpha(alpha)
			dst.DrawImage(r.dim, op)
		}
	}
}
