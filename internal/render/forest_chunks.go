package render

import (
	"image"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// Местность кешируется чанками, выровненными по миру. Чанк перерисовывается, только когда
// изменилось то, что он показывает: свет, известная земля, видимость или соседняя
// поляна. Раньше шаг по коридору перерисовывал весь кешированный экран;
// теперь перерисовываются чанки, где сдвинулся свет, а сдвиг камеры рисует только
// новый ряд чанков на краю. Выравнивание по миру к тому же держит каждый спрайт
// на том же пикселе между перерисовками, поэтому дробный масштаб не мерцает.
const (
	chunkTiles = 8
	chunkSize  = chunkTiles * TileSize
	// chunkMargin: насколько, в клетках, за пределами чанка может лежать источник
	// его содержимого: дерево высотой до 152 px, заходящее внутрь, берёт свет у своего корня,
	// а свет там смешивается и со следующим рядом клеток.
	chunkMargin = 7
	// chunkFrameBudget ограничивает время, которое кадр тратит на перерисовку устаревших чанков;
	// до этого они несколько кадров показывают прежнюю картинку.
	chunkFrameBudget = 6 * time.Millisecond
)

type forestChunk struct {
	frame   *ebiten.Image
	key     uint64
	version int // версия местности, с которой сверялся ключ
	// old: прежняя картинка, которая с fadeFrom растворяется под новой.
	old      *ebiten.Image
	fadeFrom int
}

// chunkFadeTicks: за сколько перерисованный чанк проявляется поверх старой
// картинки. Новый свет тогда расходится от героя, а не выскакивает
// квадрат за квадратом; второй раз рисуются только чанки, которые проявляются сейчас.
const chunkFadeTicks = 8

// forestChunks: кеш чанков и состояние местности, которое они рисуют.
type forestChunks struct {
	level     *domain.Level
	content   [2]uint64 // отпечаток видимых клеток, посещённые комнаты
	version   int
	view      forestView
	vis       domain.Visibility
	paths     map[domain.Point]bool
	clearings []sceneClearing
	chunks    map[image.Point]*forestChunk
	moss      map[uint64][]float32     // сетки мха по форме поляны
	plants    map[uint64]clearingHedge // последняя изгородь по форме поляны
	pool      []*ebiten.Image
	// drawn считает перерисовки чанков; budget подменяет chunkFrameBudget. Для тестов.
	drawn int
	last  image.Point
	// pending выставлен, пока устаревшие или опережающие камеру чанки ждут кадра.
	pending bool
	budget  *time.Duration
}

func (f *forestChunks) reset(level *domain.Level) {
	for _, c := range f.chunks {
		if c.old != nil {
			f.recycle(c.old)
		}
		f.recycle(c.frame)
	}
	*f = forestChunks{level: level, chunks: map[image.Point]*forestChunk{}, moss: map[uint64][]float32{}, plants: map[uint64]clearingHedge{}, pool: f.pool, drawn: f.drawn, budget: f.budget}
}

func (f *forestChunks) recycle(frame *ebiten.Image) {
	if len(f.pool) >= 48 {
		frame.Deallocate()
		return
	}
	f.pool = append(f.pool, frame)
}

func (f *forestChunks) image() *ebiten.Image {
	if n := len(f.pool); n > 0 {
		frame := f.pool[n-1]
		f.pool = f.pool[:n-1]
		return frame
	}
	return ebiten.NewImage(chunkSize, chunkSize)
}

// chunkRange перечисляет чанки, пересекающие rect.
func chunkRange(rect image.Rectangle) (lo, hi image.Point) {
	lo = image.Pt(floorDiv(rect.Min.X, chunkSize), floorDiv(rect.Min.Y, chunkSize))
	hi = image.Pt(floorDiv(rect.Max.X-1, chunkSize), floorDiv(rect.Max.Y-1, chunkSize))
	return lo, hi
}

func floorDiv(a, b int) int {
	if a < 0 {
		return -((-a + b - 1) / b)
	}
	return a / b
}

// key считает отпечаток всего, что может изменить пиксели чанка c.
func (f *forestChunks) key(c image.Point) uint64 {
	h := uint64(14695981039346656037)
	mix := func(v uint64) {
		h ^= v
		h *= 1099511628211
	}
	v := f.view
	x0, y0 := c.X*chunkTiles-chunkMargin, c.Y*chunkTiles-chunkMargin
	for y := max(0, y0); y < min(domain.Rows, y0+chunkTiles+2*chunkMargin); y++ {
		for x := max(0, x0); x < min(domain.Cols, x0+chunkTiles+2*chunkMargin); x++ {
			i := y*domain.Cols + x
			bits := uint64(math.Float32bits(v.lights[i]))
			if v.cells[i] {
				bits |= 1 << 32
			}
			if v.seen[i] {
				bits |= 1 << 33
			}
			mix(bits)
		}
	}
	rect := image.Rect(c.X*chunkSize, c.Y*chunkSize, (c.X+1)*chunkSize, (c.Y+1)*chunkSize)
	for _, cl := range f.clearings {
		if cl.reach().Overlaps(rect) {
			mix(uint64(cl.bounds.Min.X)<<48 ^ uint64(cl.bounds.Min.Y)<<32 ^ uint64(cl.bounds.Max.X)<<16 ^ uint64(cl.bounds.Max.Y))
			mix(uint64(len(cl.entrances)))
		}
	}
	return h
}

// update обновляет состояние местности, когда изменилось видимое.
func (r *Renderer) updateForest(level *domain.Level, grid domain.Grid, vis domain.Visibility, visible uint64, paths map[domain.Point]bool, visited int) {
	f := &r.forest
	if f.level != level || f.chunks == nil {
		f.reset(level)
	}
	content := [2]uint64{visible, uint64(visited)}
	if f.version > 0 && content == f.content {
		return
	}
	f.content = content
	f.version++
	f.vis = r.forestMemory.reveal(level, grid, vis)
	f.view = newForestView(grid, f.vis)
	f.paths = paths
	f.clearings = f.clearings[:0]
	for _, c := range knownClearings(f.view, paths) {
		key := c.shapeKey()
		moss := f.moss[key]
		if moss == nil {
			moss = c.mossGrid()
			f.moss[key] = moss
		}
		pk := c.plantsKey(f.view)
		hedge, ok := f.plants[key]
		if !ok || hedge.key != pk {
			hedge = clearingHedge{pk, c.plants(f.view)}
			f.plants[key] = hedge // одна запись на поляну: старая земля ушла
		}
		f.clearings = append(f.clearings, sceneClearing{c, hedge.plants, moss})
	}
}

// drawChunk рисует чанк c по текущему состоянию местности.
func (r *Renderer) drawChunk(c image.Point, chunk *forestChunk) {
	f := &r.forest
	rect := image.Rect(c.X*chunkSize, c.Y*chunkSize, (c.X+1)*chunkSize, (c.Y+1)*chunkSize)
	scene := &forestScene{level: f.level, vis: f.vis, paths: f.paths, view: f.view, props: forestProps(f.view, rect.Inset(-propReach))}
	for _, cl := range f.clearings {
		if cl.reach().Overlaps(rect) {
			scene.clearings = append(scene.clearings, cl)
		}
	}
	chunk.frame.Fill(Black)
	r.drawForestRegion(chunk.frame, scene, rect, rect.Min.X, rect.Min.Y)
	chunk.key, chunk.version = f.key(c), f.version
	f.drawn++
	f.last = c
}

// drawCachedForest рисует местность под областью просмотра (пиксели мира) в dst.
// visible: это visibleSignature(vis.Visible), считается один раз за ход.
func (r *Renderer) drawCachedForest(dst *ebiten.Image, level *domain.Level, grid domain.Grid, vis domain.Visibility, visible uint64, paths map[domain.Point]bool, visited int, viewport image.Rectangle) {
	r.updateForest(level, grid, vis, visible, paths, visited)
	f := &r.forest
	lo, hi := chunkRange(viewport)
	inView := func(c image.Point) bool { return c.X >= lo.X && c.X <= hi.X && c.Y >= lo.Y && c.Y <= hi.Y }
	// Чанки на экране, которым ещё нечего показать, рисуются сразу. Устаревшие
	// и кольцо, готовящееся впереди камеры, ждут в очереди.
	var queue []image.Point
	for cy := lo.Y - 1; cy <= hi.Y+1; cy++ {
		for cx := lo.X - 1; cx <= hi.X+1; cx++ {
			c := image.Pt(cx, cy)
			chunk := f.chunks[c]
			switch {
			case chunk == nil && inView(c):
				chunk = &forestChunk{frame: f.image()}
				f.chunks[c] = chunk
				r.drawChunk(c, chunk)
			case chunk == nil && !c.In(worldChunks):
				// Камера никогда не показывает ничего за картой: там готовить нечего.
			case chunk == nil:
				queue = append(queue, c)
			case chunk.version != f.version:
				if f.key(c) == chunk.key {
					chunk.version = f.version // здесь без изменений
				} else {
					queue = append(queue, c)
				}
			}
		}
	}
	// Сначала экран, от его середины (героя) к краям, чтобы после
	// бега вид вокруг героя устоялся раньше краёв.
	center := viewport.Min.Add(viewport.Size().Div(2))
	distance := func(c image.Point) int {
		d := image.Pt(c.X*chunkSize+chunkSize/2, c.Y*chunkSize+chunkSize/2).Sub(center)
		n := d.X*d.X + d.Y*d.Y
		if !inView(c) {
			n += 1 << 40
		}
		return n
	}
	sort.Slice(queue, func(i, j int) bool { return distance(queue[i]) < distance(queue[j]) })
	// Перерисовываем, пока не кончится бюджет времени кадра, минимум один чанк:
	// быстрая машина устанавливает новый вид за кадр-два, медленная
	// держит кадры короткими и успокаивается ещё за несколько.
	// После бега или прыжка большая часть экрана устарела: тогда минимум два
	// чанка за кадр, чтобы и медленная машина успокоилась за несколько кадров.
	minimum, stale := 1, 0
	for _, c := range queue {
		if inView(c) {
			stale++
		}
	}
	if stale > 4 && r.forest.budget == nil {
		minimum = 2
	}
	start := time.Now()
	f.pending = false
	for i, c := range queue {
		if i >= minimum && time.Since(start) >= r.chunkBudget() {
			f.pending = true // ещё в следующем кадре, даже если больше ничего не движется
			break
		}
		chunk := f.chunks[c]
		if chunk == nil {
			chunk = &forestChunk{frame: f.image()}
			f.chunks[c] = chunk
		} else {
			// Оставляем картинку на экране и проявляем новую поверх.
			if chunk.old != nil {
				f.recycle(chunk.old)
			}
			chunk.old, chunk.frame, chunk.fadeFrom = chunk.frame, f.image(), r.tick
		}
		r.drawChunk(c, chunk)
	}
	for c, chunk := range f.chunks {
		if c.X < lo.X-2 || c.X > hi.X+2 || c.Y < lo.Y-2 || c.Y > hi.Y+2 {
			if chunk.old != nil {
				f.recycle(chunk.old)
			}
			f.recycle(chunk.frame)
			delete(f.chunks, c)
		}
	}
	for cy := lo.Y; cy <= hi.Y; cy++ {
		for cx := lo.X; cx <= hi.X; cx++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(cx*chunkSize-viewport.Min.X), float64(cy*chunkSize-viewport.Min.Y))
			chunk := f.chunks[image.Pt(cx, cy)]
			dst.DrawImage(chunk.frame, op)
			if chunk.old == nil {
				continue
			}
			if t := float32(r.tick-chunk.fadeFrom) / chunkFadeTicks; t < 1 {
				op.ColorScale.ScaleAlpha(1 - t)
				dst.DrawImage(chunk.old, op)
			} else {
				f.recycle(chunk.old)
				chunk.old = nil
			}
		}
	}
}

// chunkBudget: сколько кадр может тратить на перерисовку устаревших чанков.
func (r *Renderer) chunkBudget() time.Duration {
	if r.forest.budget != nil {
		return *r.forest.budget
	}
	return chunkFrameBudget
}

// fading сообщает, проявляется ли ещё перерисованный чанк.
func (f *forestChunks) fading(tick int) bool {
	for _, c := range f.chunks {
		if c.old != nil && tick-c.fadeFrom < chunkFadeTicks {
			return true
		}
	}
	return false
}

// clearingHedge: изгородь поляны и plantsKey, для которого она построена.
type clearingHedge struct {
	key    uint64
	plants []forestProp
}

// worldChunks: диапазон чанков, покрывающих карту.
var worldChunks = func() image.Rectangle {
	lo, hi := chunkRange(image.Rect(0, 0, domain.Cols*TileSize, domain.Rows*TileSize))
	return image.Rectangle{Min: lo, Max: hi.Add(image.Pt(1, 1))}
}()
