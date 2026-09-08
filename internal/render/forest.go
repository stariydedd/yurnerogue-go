package render

import (
	"image"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// Scenery uses only revealed terrain. Unknown rooms must look exactly like
// ordinary forest: their geometry must not influence tree placement or light.
type forestView struct {
	ground   map[domain.Point]bool
	distance map[domain.Point]int
}

var forestDirs = []domain.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}

type forestCacheKey struct {
	level    *domain.Level
	player   domain.Point
	visited  int
	viewport image.Rectangle
}

type forestCache struct {
	key   forestCacheKey
	frame *ebiten.Image
}

// Terrain is static between player steps. Cache its GPU image instead of
// rebuilding the forest and its low foliage on every animation frame.
func (r *Renderer) drawCachedForest(dst *ebiten.Image, grid domain.Grid, vis domain.Visibility, paths map[domain.Point]bool, key forestCacheKey) {
	if r.forest == nil || r.forest.key != key {
		var frame *ebiten.Image
		if r.forest != nil {
			frame = r.forest.frame
			if frame.Bounds().Size() != key.viewport.Size() {
				frame.Deallocate()
				frame = nil
			}
		}
		if frame == nil {
			frame = ebiten.NewImage(key.viewport.Dx(), key.viewport.Dy())
		}
		frame.Fill(Black)
		r.drawForestTerrain(frame, grid, vis, paths, key.viewport.Min.X, key.viewport.Min.Y)
		r.forest = &forestCache{key: key, frame: frame}
	}
	dst.DrawImage(r.forest.frame, nil)
}

func newForestView(grid domain.Grid, vis domain.Visibility) forestView {
	v := forestView{ground: map[domain.Point]bool{}, distance: map[domain.Point]int{}}
	var queue []domain.Point
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if (vis.Visible[p] || vis.Explored[p]) && isFloor(grid.At(x, y)) {
				v.ground[p] = true
				if vis.Visible[p] {
					v.distance[p] = 0
					queue = append(queue, p)
				}
			}
		}
	}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		d := v.distance[p]
		if d >= 7 {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				q := domain.Point{X: p.X + dx, Y: p.Y + dy}
				if _, ok := v.distance[q]; !ok && domain.InBounds(q.X, q.Y) {
					v.distance[q] = d + 1
					queue = append(queue, q)
				}
			}
		}
	}
	return v
}

// A few leaf/root pixels may cross the edge, never the centre of a walkable
// cell. This also keeps a one-cell corridor open along its full length.
func (v forestView) fits(rect image.Rectangle, fringe int) bool {
	for y := max(0, rect.Min.Y/TileSize-1); y <= min(domain.Rows-1, rect.Max.Y/TileSize); y++ {
		for x := max(0, rect.Min.X/TileSize-1); x <= min(domain.Cols-1, rect.Max.X/TileSize); x++ {
			if v.ground[domain.Point{X: x, Y: y}] && rect.Overlaps(image.Rect(x*TileSize, y*TileSize, (x+1)*TileSize, (y+1)*TileSize).Inset(fringe)) {
				return false
			}
		}
	}
	return true
}

func (v forestView) light(rect image.Rectangle) float32 {
	distance := 8
	for y := max(0, rect.Min.Y/TileSize); y <= min(domain.Rows-1, rect.Max.Y/TileSize); y++ {
		for x := max(0, rect.Min.X/TileSize); x <= min(domain.Cols-1, rect.Max.X/TileSize); x++ {
			if d, ok := v.distance[domain.Point{X: x, Y: y}]; ok {
				distance = min(distance, d)
			}
		}
	}
	return [...]float32{1, .98, .9, .76, .58, .43, .32, .25, .2}[distance]
}

type forestProp struct {
	role        string
	frame       int
	rect        image.Rectangle
	rotation    int
	groundCover bool
	lightAnchor image.Rectangle
}

// A rooted object and its moss share light sampled at their contact with the
// ground, not at whichever crown pixel happens to be closest to the clearing.
func forestFoot(p forestProp) image.Rectangle {
	x, y := (p.rect.Min.X+p.rect.Max.X)/2, p.rect.Max.Y-4
	return image.Rect(x-p.rect.Dx()/4, y-6, x+p.rect.Dx()/4, y+6)
}

func forestBed(p forestProp) forestProp {
	foot := forestFoot(p)
	x, y := (foot.Min.X+foot.Max.X)/2, (foot.Min.Y+foot.Max.Y)/2
	w := p.rect.Dx() + 24
	return forestProp{
		role: "moss", frame: p.frame, rect: image.Rect(x-w/2, y-w/4, x+w/2, y+w/4),
		groundCover: true, lightAnchor: foot,
	}
}

func forestRootGrass(p forestProp) forestProp {
	return forestProp{role: "verge", frame: p.frame,
		rect:        image.Rect(p.rect.Min.X+p.rect.Dx()/5, p.rect.Max.Y-14, p.rect.Max.X-p.rect.Dx()/5, p.rect.Max.Y),
		lightAnchor: forestFoot(p)}
}

func (p forestProp) lightingBounds() image.Rectangle {
	if !p.lightAnchor.Empty() {
		return p.lightAnchor
	}
	return p.rect
}

// Crown and trunk bounds omit transparent sprite margins, allowing interlocking
// groves without hiding trunks behind foreground crowns.
func forestTreeSilhouette(p forestProp) (image.Rectangle, image.Rectangle) {
	// The four packed trees share a root baseline but have different crown tops.
	tops := [...]int{8, 4, 22, 34}
	frame := ((p.frame % 4) + 4) % 4
	crown := image.Rect(p.rect.Min.X+5, p.rect.Min.Y+p.rect.Dy()*tops[frame]/128,
		p.rect.Max.X-5, p.rect.Min.Y+p.rect.Dy()*88/128)
	trunk := image.Rect(p.rect.Min.X+p.rect.Dx()/3, crown.Max.Y,
		p.rect.Max.X-p.rect.Dx()/3, p.rect.Max.Y-8)
	return crown, trunk
}

// Edge trees and woodland trees share this one pass.
// Select in world space before viewport culling so panning cannot reshuffle it.
func forestTrees(v forestView) []forestProp {
	type candidate struct {
		prop     forestProp
		priority int
	}
	var candidates []candidate
	add := func(x, y, h, priority int) {
		w, height := 100+(h/113)%21, 128+(h/199)%25
		rect := image.Rect(x-w/2, y-height, x+w/2, y)
		if v.fits(rect, 5) {
			candidates = append(candidates, candidate{forestProp{role: "tree", frame: h / 7, rect: rect}, priority})
		}
	}
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			h := cellHash(x*5+3, y)
			if v.ground[domain.Point{X: x, Y: y}] && !v.ground[domain.Point{X: x, Y: y - 1}] && h%2 == 0 {
				add(x*TileSize+16+h%13-6, y*TileSize+4-(h/13)%19, h, h%10000)
			}
		}
	}
	for gy := 0; gy <= (domain.Rows*TileSize+160)/56; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+160)/56; gx++ {
			h := cellHash(gx+317, gy+911)
			if h%4 == 0 {
				continue
			}
			x := gx*56 + (gy%2)*28 + h%41 - 20
			y := gy*56 + (h/41)%43 - 21
			add(x, y, h, 10000+h%100000)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].priority < candidates[j].priority })
	var trees []forestProp
	space := forestSpace{}
	for _, c := range candidates {
		crown, trunk := forestTreeSilhouette(c.prop)
		if space.free(crown.Inset(-6)) && space.free(trunk.Inset(-6)) {
			trees = append(trees, c.prop)
			space.occupy(crown)
			space.occupy(trunk)
		}
	}
	return trees
}

// Spatial buckets keep full-world selection cheap even on large viewports.
// Buckets are only queried, never iterated as a map: ordering is deterministic.
type forestSpace map[image.Point][]image.Rectangle

func (s forestSpace) free(rect image.Rectangle) bool {
	for y := rect.Min.Y / 64; y <= (rect.Max.Y-1)/64; y++ {
		for x := rect.Min.X / 64; x <= (rect.Max.X-1)/64; x++ {
			for _, occupied := range s[image.Pt(x, y)] {
				if rect.Overlaps(occupied) {
					return false
				}
			}
		}
	}
	return true
}

func (s forestSpace) occupy(rect image.Rectangle) {
	for y := rect.Min.Y / 64; y <= (rect.Max.Y-1)/64; y++ {
		for x := rect.Min.X / 64; x <= (rect.Max.X-1)/64; x++ {
			p := image.Pt(x, y)
			s[p] = append(s[p], rect)
		}
	}
}

// Low foliage may surround the narrow trunk but must stay below the crown.
// The protected silhouette keeps plants from reading as trees piled together.
func (s forestSpace) protectTree(p forestProp) {
	crown, trunk := forestTreeSilhouette(p)
	s.occupy(crown)
	s.occupy(trunk)
}

func forestProps(v forestView, viewport image.Rectangle) []forestProp {
	// All tall/solid props participate in spacing even outside the viewport.
	// Verge is ground cover and intentionally overlaps roots and other fringes.
	scenery := forestTrees(v)
	space := forestSpace{}
	for _, tree := range scenery {
		space.protectTree(tree)
	}
	var props []forestProp
	add := func(role string, frame int, rect image.Rectangle) bool {
		if v.fits(rect, 5) && space.free(rect) {
			scenery = append(scenery, forestProp{role: role, frame: frame, rect: rect})
			inset := 5
			if role == "bush" && rect.Dy() > 30 {
				inset = 10
			}
			space.occupy(rect.Inset(inset))
			return true
		}
		return false
	}
	// Boundary props are generated in coordinate order, never map iteration
	// order, so depth ties and variants remain stable between frames.
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if !v.ground[p] {
				continue
			}
			for i, d := range forestDirs {
				if v.ground[domain.Point{X: x + d.X, Y: y + d.Y}] {
					continue
				}
				h := cellHash(x*5+i, y)
				ex, ey := x*TileSize+16+d.X*16, y*TileSize+16+d.Y*16
				// A low continuous fringe joins isolated bushes and roots to
				// the floor. Its rotated bounds obey the same corridor guard.
				vw, vh, rotation := 64, 32, 0
				switch {
				case d.X == 1:
					vw, vh, rotation = 32, 64, 1
				case d.X == -1:
					vw, vh, rotation = 32, 64, 3
				case d.Y == 1:
					rotation = 2
				}
				// Let the foliage follow the same uneven bank as the ground,
				// instead of rebuilding a perfectly straight border over it.
				along, boundary := ex, ey
				if d.X != 0 {
					along, boundary = ey, ex
				}
				bank := max(0, int(groundWave(along, boundary, i)))
				vx, vy := ex+d.X*(vw/2-5+bank), ey+d.Y*(vh/2-5+bank)
				vr := image.Rect(vx-vw/2, vy-vh/2, vx+vw/2, vy+vh/2)
				if vr.Overlaps(viewport) && v.fits(vr, 5) {
					props = append(props, forestProp{role: "verge", frame: h, rect: vr, rotation: rotation})
				}
				w, hgt := 48+h%21, 40+(h/17)%17
				role := "bush"
				frame := (h / 5) % 3
				if (h/29)%13 == 0 {
					frame = 3
				} // Cyan plants are accents, not a border pattern.
				if h%4 == 0 {
					role = "ruins"
					w, hgt = 48, 60
					frame = (h / 5) % 3 // Mostly broken blocks, fewer pillars.
				}
				ox, oy := d.X*(w/2-4), d.Y*(hgt/2-4)
				tangent := h%11 - 5
				cx, cy := ex+ox+d.Y*tangent, ey+oy+d.X*tangent
				add(role, frame, image.Rect(cx-w/2, cy-hgt/2, cx+w/2, cy+hgt/2))
			}
		}
	}
	// Reserve gaps for masonry before large bushes consume them. Broken blocks
	// and occasional smaller rubble form loose clusters, never continuous walls.
	for gy := 0; gy <= (domain.Rows*TileSize+96)/72; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+96)/80; gx++ {
			h := cellHash(gx+419, gy+827)
			if h%5 == 0 {
				continue
			}
			x, y := gx*80+(gy%2)*40+h%49-24, gy*72+(h/49)%41-20
			w, height := 44+(h/71)%17, 42+(h/137)%13
			frame := 0
			if h%3 == 0 {
				frame = 2
				height += 12 // Fallen-block art includes headroom above the rubble.
			}
			if h%13 == 0 {
				frame = 1
				height = 72
			}
			if add("ruins", frame, image.Rect(x-w/2, y-height, x+w/2, y)) && h%3 == 0 {
				side := 1
				if h%2 == 0 {
					side = -1
				}
				cx, cy := x+side*(w/2+12), y+6-(h/17)%13
				add("ruins", 2, image.Rect(cx-17, cy-40, cx+17, cy))
			}
		}
	}
	// Compose clusters around existing anchors: broad leaves, then smaller
	// companion ferns. Roots and masonry stay readable between the plants.
	anchors := append([]forestProp(nil), scenery...)
	for _, anchor := range anchors {
		if anchor.role != "tree" && anchor.role != "ruins" {
			continue
		}
		x, y := (anchor.rect.Min.X+anchor.rect.Max.X)/2, anchor.rect.Max.Y
		h := cellHash(x+271, y+613)
		for i, side := range []int{-1, 1} {
			w, height := 48+(h/31+i*7)%19, 38+(h/73+i*5)%15
			cx, cy := x+side*(anchor.rect.Dx()/3+8+(h/19)%13), y+12+(h/43)%13
			frame := (h/7 + i) % 3
			if anchor.role == "ruins" && (h+i)%7 == 0 {
				frame = 3
			}
			add("bush", frame, image.Rect(cx-w/2, cy-height, cx+w/2, cy))
		}
	}
	// Low, asymmetric groups occupy gaps between trees, not just the clearing
	// border. Stone clusters interrupt the green without becoming a new wall.
	for gy := 0; gy <= (domain.Rows*TileSize+96)/28; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+96)/32; gx++ {
			h := cellHash(gx+701, gy+239)
			if h%5 == 0 {
				continue
			}
			x := gx*32 + (gy%2)*16 + h%29 - 14
			y := gy*28 + (h/29)%27 - 13
			role, frame := "bush", (h/11)%3
			w, height := 54+(h/31)%29, 44+(h/73)%25
			if h%6 == 0 {
				role, frame = "ruins", ((h/17)%2)*2
				w, height = 46+(h/31)%11, 58+(h/73)%13
			} else if h%47 == 0 {
				frame = 3
			}
			add(role, frame, image.Rect(x-w/2, y-height, x+w/2, y))
		}
	}
	// A separate small-plant pass fills the remaining pockets without enlarging
	// the background leaf carpet. Ferns and occasional cyan shoots stay readable
	// beside masonry and below trees, with the same trunk and corridor guards.
	for gy := 0; gy <= (domain.Rows*TileSize+48)/24; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+48)/28; gx++ {
			h := cellHash(gx+1019, gy+367)
			if h%3 == 0 {
				continue
			}
			x, y := gx*28+(gy%2)*14+h%19-9, gy*24+(h/19)%17-8
			w, height := 20+(h/43)%13, 18+(h/101)%13
			frame := 1 + (h/7)%2
			if h%11 == 0 {
				frame = 3
			}
			add("bush", frame, image.Rect(x-w/2, y-height, x+w/2, y))
		}
	}
	for _, p := range scenery {
		if p.role == "tree" || p.role == "ruins" {
			p.lightAnchor = forestFoot(p)
			bed := forestBed(p)
			if bed.rect.Overlaps(viewport) {
				props = append(props, bed)
			}
		}
		if p.rect.Overlaps(viewport) {
			props = append(props, p)
		}
	}
	// A shaded low understory connects the separated silhouettes. It is drawn
	// BEHIND trunks and stones, never on top of them like another row of crowns.
	area := viewport.Inset(-96)
	for gy := max(0, area.Min.Y/24); gy <= area.Max.Y/24; gy++ {
		for gx := max(0, area.Min.X/28); gx <= area.Max.X/28; gx++ {
			h := cellHash(gx+127, gy+563)
			if h%11 == 0 {
				continue
			}
			x, y := gx*28+(gy%2)*14+h%21-10, gy*24+(h/25)%21-10
			w, height := 46+(h/53)%25, 38+(h/97)%21
			rect := image.Rect(x-w/2, y-height, x+w/2, y)
			if rect.Overlaps(viewport) && v.fits(rect, 5) {
				frame := 1 + (h/7)%2
				if h%7 == 0 {
					frame = 0
				}
				props = append(props, forestProp{role: "bush", frame: frame, rect: rect, groundCover: true})
			}
		}
	}
	sort.SliceStable(props, func(i, j int) bool {
		if props[i].groundCover != props[j].groundCover {
			return props[i].groundCover
		}
		if (props[i].role == "moss") != (props[j].role == "moss") {
			return props[j].role == "moss"
		}
		if (props[i].role == "verge") != (props[j].role == "verge") {
			return props[i].role == "verge"
		}
		return props[i].rect.Max.Y < props[j].rect.Max.Y
	})
	return props
}

func (r *Renderer) drawForestTerrain(dst *ebiten.Image, grid domain.Grid, vis domain.Visibility, paths map[domain.Point]bool, camX, camY int) {
	viewport := image.Rect(camX, camY, camX+r.Layout.GridW, camY+r.Layout.GridH)
	v := newForestView(grid, vis)
	props := forestProps(v, viewport)
	// Continuous shaded grass/soil below the forest. This is decorative ground,
	// unrelated to hidden rooms, rather than a second layer of floating crowns.
	for y := camY / TileSize; y <= viewport.Max.Y/TileSize; y++ {
		for x := camX / TileSize; x <= viewport.Max.X/TileSize; x++ {
			rect := image.Rect(x*TileSize, y*TileSize, (x+1)*TileSize, (y+1)*TileSize)
			r.drawForestSoil(dst, v, rect, camX, camY, cellHash(x, y))
		}
	}
	// Dense low foliage goes below root beds, not around large exclusion halos.
	// Moss then restores the grounded contact point without clearing a gap in
	// the surrounding vegetation. Both remain below opaque walkable tiles.
	for _, prop := range props {
		if prop.groundCover && prop.role == "bush" {
			r.drawForestProp(dst, prop, camX, camY, v.light(prop.lightingBounds())*.66)
		}
	}
	for _, prop := range props {
		if prop.role == "moss" {
			r.drawForestProp(dst, prop, camX, camY, v.light(prop.lightingBounds())*.85)
			r.drawForestContact(dst, prop.lightAnchor, camX, camY)
		}
	}
	// Include cells just outside the camera whose decorative bank reaches in.
	for y := max(0, (camY-10)/TileSize); y <= min(domain.Rows-1, (viewport.Max.Y+10)/TileSize); y++ {
		for x := max(0, (camX-10)/TileSize); x <= min(domain.Cols-1, (viewport.Max.X+10)/TileSize); x++ {
			p := domain.Point{X: x, Y: y}
			if !v.ground[p] {
				continue
			}
			role := "floor"
			if paths[p] {
				role = "path"
			}
			h := cellHash(x, y)
			r.drawWalkableGround(dst, v, paths, p, camX, camY, vis.Visible[p])
			// Sparse low plants in the clearing, never a second obstacle grid.
			if role == "floor" && h%31 == 0 && vis.Visible[p] {
				r.drawTile(dst, "decor", x, y, camX, camY, h/31)
			}
		}
	}
	r.drawClearingBanks(dst, v, vis, paths, camX, camY)
	for _, prop := range props {
		if prop.groundCover {
			continue
		}
		light := v.light(prop.lightingBounds())
		r.drawForestProp(dst, prop, camX, camY, light)
	}
}

// Interpolate light between shared tile corners. Uniform per-tile brightness
// would expose a checkerboard on the quiet soil between plants.
func (r *Renderer) drawForestSoil(dst *ebiten.Image, v forestView, rect image.Rectangle, camX, camY, frame int) {
	img := r.sprites.Frame("floor", frame)
	if img == nil {
		return
	}
	b := img.Bounds()
	points := [4]image.Point{rect.Min, {X: rect.Max.X, Y: rect.Min.Y}, {X: rect.Min.X, Y: rect.Max.Y}, rect.Max}
	sources := [4]image.Point{b.Min, {X: b.Max.X, Y: b.Min.Y}, {X: b.Min.X, Y: b.Max.Y}, b.Max}
	var vertices [4]ebiten.Vertex
	for i, p := range points {
		light := v.light(image.Rect(p.X, p.Y, p.X+1, p.Y+1)) * .62
		vertices[i] = ebiten.Vertex{DstX: float32(p.X - camX), DstY: float32(p.Y - camY),
			SrcX: float32(sources[i].X), SrcY: float32(sources[i].Y),
			ColorR: light, ColorG: light, ColorB: light, ColorA: 1}
	}
	dst.DrawTriangles(vertices[:], []uint16{0, 1, 2, 1, 3, 2}, img, nil)
}

// A small soft contact shadow sits on the ground immediately beneath roots,
// not offset like a floating object's shadow. It is rendered before paths.
func (r *Renderer) drawForestContact(dst *ebiten.Image, foot image.Rectangle, camX, camY int) {
	const segments = 24
	var vertices [segments + 1]ebiten.Vertex
	var indices [segments * 3]uint16
	x, y := float32(foot.Min.X+foot.Max.X)/2-float32(camX), float32(foot.Min.Y+foot.Max.Y)/2-float32(camY)
	vertices[0] = ebiten.Vertex{DstX: x, DstY: y, SrcX: 1, SrcY: 1, ColorA: .62}
	for i := 0; i < segments; i++ {
		a := float64(i) * 2 * math.Pi / segments
		vertices[i+1] = ebiten.Vertex{DstX: x + float32(math.Cos(a))*float32(foot.Dx())*.64,
			DstY: y + float32(math.Sin(a))*float32(foot.Dy())*.8, SrcX: 1, SrcY: 1}
		indices[i*3], indices[i*3+1], indices[i*3+2] = 0, uint16(i+1), uint16((i+1)%segments+1)
	}
	dst.DrawTriangles(vertices[:], indices[:], r.dim, nil)
}

func (r *Renderer) drawForestProp(dst *ebiten.Image, p forestProp, camX, camY int, light float32) {
	img := r.sprites.Frame(p.role, p.frame)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	w, h := float64(p.rect.Dx()), float64(p.rect.Dy())
	if p.rotation%2 != 0 {
		w, h = h, w
	}
	op.GeoM.Scale(w/float64(img.Bounds().Dx()), h/float64(img.Bounds().Dy()))
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(float64(p.rotation) * math.Pi / 2)
	op.GeoM.Translate(float64(p.rect.Min.X-camX)+float64(p.rect.Dx())/2, float64(p.rect.Min.Y-camY)+float64(p.rect.Dy())/2)
	op.ColorScale.Scale(light, light, light, 1)
	if p.role == "moss" {
		op.ColorScale.ScaleAlpha(.72)
	}
	dst.DrawImage(img, op)
	if p.role == "tree" {
		// A few foreground blades cover the bottom root pixels; the rest of
		// the trunk stays clear. This fringe fits entirely inside the tree.
		r.drawForestProp(dst, forestRootGrass(p), camX, camY, light*.86)
	}
}
