package render

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// The terrain is cached in world-aligned chunks. A chunk is redrawn only when
// something it shows has changed: light, known ground, visibility or a nearby
// clearing. A step along a corridor used to redraw the whole cached screen;
// now it redraws the chunks whose light moved, and a camera slide only draws
// the new row of chunks at the edge. World alignment also keeps every sprite
// at the same pixel between redraws, so fractional scaling cannot shimmer.
const (
	chunkTiles = 8
	chunkSize  = chunkTiles * TileSize
	// chunkMargin is how far, in tiles, outside a chunk its content can come
	// from: tall props reach in and sample light under their full height.
	chunkMargin = 5
	// chunksPerFrame bounds redraws of chunks that are already on screen;
	// until then they keep their previous picture for a few frames.
	chunksPerFrame = 3
)

type forestChunk struct {
	frame   *ebiten.Image
	key     uint64
	version int // terrain version the key was checked against
}

// forestChunks is the chunk cache and the terrain state its chunks draw.
type forestChunks struct {
	level     *domain.Level
	content   [2]uint64 // visible cells signature, visited rooms
	version   int
	view      forestView
	vis       domain.Visibility
	paths     map[domain.Point]bool
	clearings []sceneClearing
	chunks    map[image.Point]*forestChunk
	moss      map[uint64][]float32 // moss grids by clearing shape
	pool      []*ebiten.Image
	// drawn counts chunk redraws, for tests.
	drawn int
}

func (f *forestChunks) reset(level *domain.Level) {
	for _, c := range f.chunks {
		f.recycle(c.frame)
	}
	*f = forestChunks{level: level, chunks: map[image.Point]*forestChunk{}, moss: map[uint64][]float32{}, pool: f.pool, drawn: f.drawn}
}

func (f *forestChunks) recycle(frame *ebiten.Image) {
	if len(f.pool) >= 16 {
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

// chunkRange lists the chunks that intersect rect.
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

// key fingerprints everything that can change the pixels of chunk c.
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

// update refreshes the terrain state when what is visible has changed.
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
		f.clearings = append(f.clearings, sceneClearing{c, c.plants(f.view), moss})
	}
}

// drawChunk renders chunk c from the current terrain state.
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
}

// drawCachedForest draws the terrain under viewport (world pixels) into dst.
// visible is visibleSignature(vis.Visible), computed once per turn.
func (r *Renderer) drawCachedForest(dst *ebiten.Image, level *domain.Level, grid domain.Grid, vis domain.Visibility, visible uint64, paths map[domain.Point]bool, visited int, viewport image.Rectangle) {
	r.updateForest(level, grid, vis, visible, paths, visited)
	f := &r.forest
	lo, hi := chunkRange(viewport)
	// One ring of chunks around the view is prepared ahead of the camera.
	budget := chunksPerFrame
	for ring := 0; ring <= 1; ring++ {
		for cy := lo.Y - ring; cy <= hi.Y+ring; cy++ {
			for cx := lo.X - ring; cx <= hi.X+ring; cx++ {
				if ring == 1 && cx >= lo.X && cx <= hi.X && cy >= lo.Y && cy <= hi.Y {
					continue
				}
				c := image.Pt(cx, cy)
				chunk := f.chunks[c]
				switch {
				case chunk == nil && ring == 0:
					// Nothing to show there yet: draw it now.
					chunk = &forestChunk{frame: f.image()}
					f.chunks[c] = chunk
					r.drawChunk(c, chunk)
				case chunk == nil && budget > 0:
					chunk = &forestChunk{frame: f.image()}
					f.chunks[c] = chunk
					r.drawChunk(c, chunk)
					budget--
				case chunk != nil && chunk.version != f.version:
					if f.key(c) == chunk.key {
						chunk.version = f.version // unchanged here
					} else if budget > 0 {
						r.drawChunk(c, chunk)
						budget--
					}
				}
			}
		}
	}
	for c, chunk := range f.chunks {
		if c.X < lo.X-2 || c.X > hi.X+2 || c.Y < lo.Y-2 || c.Y > hi.Y+2 {
			f.recycle(chunk.frame)
			delete(f.chunks, c)
		}
	}
	for cy := lo.Y; cy <= hi.Y; cy++ {
		for cx := lo.X; cx <= hi.X; cx++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(cx*chunkSize-viewport.Min.X), float64(cy*chunkSize-viewport.Min.Y))
			dst.DrawImage(f.chunks[image.Pt(cx, cy)].frame, op)
		}
	}
}
