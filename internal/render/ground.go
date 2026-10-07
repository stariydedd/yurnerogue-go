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

// Общие волны в координатах мира не дают свежей выемки на каждой границе клеток.
// Пол может немного заходить в лес, но никогда не прячет центральные
// 22x22px проходимой клетки. Столкновения и скрытые данные карты не используются.
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
		edge[side] = !v.isGround(p.X+d.X, p.Y+d.Y)
	}
	origin := image.Pt(p.X*TileSize, p.Y*TileSize)
	if edge == [4]bool{} {
		return []groundSpan{{rect: image.Rect(0, 0, TileSize, TileSize).Add(origin)}}
	}
	// Квантованный двухпиксельный контур повторяет арт, с закруглёнными выпуклыми углами.
	state := func(x, y int) int {
		if x < 0 && !edge[1] || x >= TileSize && !edge[0] || y < 0 && !edge[3] || y >= TileSize && !edge[2] {
			return -1
		}
		wx, wy := origin.X+x, origin.Y+y
		owner := domain.Point{X: int(math.Floor(float64(wx) / TileSize)), Y: int(math.Floor(float64(wy) / TileSize))}
		if owner != p && v.isGround(owner.X, owner.Y) {
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
		// Сохраняем ту же гарантию центра, что у твёрдых предметов леса, включая
		// весь отсчёт 2px, если он касается защищённого центра.
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

// Сплошная поверхность 256px хранится как четыре квадранта по 128px. Выборка
// вместо сжатия целой картинки в каждую клетку сохраняет детали
// и непрерывность между клетками, поворотами, краями экрана и движением камеры.
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

// Узкие неровные полосы травы смягчают открытые края троп, оставляя
// как видимую тропу хотя бы центральные 22x22 пикселя каждой клетки коридора.
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
		// На обработку края влияют только открытые соседи; скрытая геометрия
		// проходов не должна менять вид видимого конца тропы.
		if v.ground[q] && paths[q] {
			continue
		}
		strips = append(strips, trailFringe(p, side)...)
	}
	return strips
}

// r.dim уже содержит альфу ExploredDim. Тень касания складываем
// с этой непрозрачностью, а не умножаем альфу тумана второй раз.
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

// groundPad: насколько контур земли клетки выходит за саму клетку.
const groundPad = 10

// groundCache хранит землю, контур и тень края каждой проходимой клетки
// маленькими картинками. Контур рисуется сотней полосок по 2 px на клетку края;
// рисовать их при каждой перерисовке чанка было самой дорогой её частью. У клетки
// один слот для видимого вида и один для запомненного, поэтому
// вход в поле зрения и выход из него используют оба, а слот перерисовывается на месте
// только когда новые известные соседи меняют контур. Памяти уходит две
// картинки на известную клетку, а не больше с каждым вариантом.
type groundCache struct {
	level *domain.Level
	cells map[groundSlot]*groundCell
}

type groundSlot struct {
	p       domain.Point
	visible bool
}

type groundCell struct {
	neighbors uint16 // известная земля в блоке 3x3 вокруг клетки
	img       *ebiten.Image
}

func groundNeighbors(v forestView, p domain.Point) uint16 {
	var mask uint16
	for i, d := range [9]domain.Point{{X: -1, Y: -1}, {Y: -1}, {X: 1, Y: -1}, {X: -1}, {}, {X: 1}, {X: -1, Y: 1}, {Y: 1}, {X: 1, Y: 1}} {
		if v.isGround(p.X+d.X, p.Y+d.Y) {
			mask |= 1 << i
		}
	}
	return mask
}

// drawCachedGround рисует проходимую землю p так же, как drawWalkableGround.
func (r *Renderer) drawCachedGround(dst *ebiten.Image, level *domain.Level, v forestView, paths map[domain.Point]bool, p domain.Point, camX, camY int, visible bool) {
	c := &r.ground
	if c.level != level || c.cells == nil {
		for _, cell := range c.cells {
			cell.img.Deallocate()
		}
		*c = groundCache{level: level, cells: map[groundSlot]*groundCell{}}
	}
	slot := groundSlot{p: p, visible: visible}
	neighbors := groundNeighbors(v, p)
	cell := c.cells[slot]
	switch {
	case cell == nil:
		cell = &groundCell{neighbors: neighbors, img: ebiten.NewImage(TileSize+2*groundPad, TileSize+2*groundPad)}
		c.cells[slot] = cell
		r.drawWalkableGround(cell.img, v, paths, p, p.X*TileSize-groundPad, p.Y*TileSize-groundPad, visible)
	case cell.neighbors != neighbors:
		cell.neighbors = neighbors
		cell.img.Clear()
		r.drawWalkableGround(cell.img, v, paths, p, p.X*TileSize-groundPad, p.Y*TileSize-groundPad, visible)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.X*TileSize-groundPad-camX), float64(p.Y*TileSize-groundPad-camY))
	dst.DrawImage(cell.img, op)
}
