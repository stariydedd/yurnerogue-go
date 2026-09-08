package render

import (
	"image"
	"reflect"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func clearingFixture() (forestView, map[domain.Point]bool) {
	v := forestView{ground: map[domain.Point]bool{}}
	for y := 6; y < 15; y++ {
		for x := 8; x < 24; x++ {
			v.ground[domain.Point{X: x, Y: y}] = true
		}
	}
	paths := map[domain.Point]bool{}
	for x := 24; x < 30; x++ {
		p := domain.Point{X: x, Y: 10}
		v.ground[p] = true
		paths[p] = true
	}
	return v, paths
}

func TestClearingUsesRoomScaleLobesAndLeavesQuietCentre(t *testing.T) {
	v, paths := clearingFixture()
	cs := knownClearings(v, paths)
	if len(cs) != 1 {
		t.Fatalf("clearings: %d", len(cs))
	}
	c := cs[0]
	if len(c.cells) != 16*9 || len(c.entrances) != 1 {
		t.Fatal("incorrect known floor or entrances")
	}
	if c.distance(272, 208) >= 0 || c.distance(752, 464) >= 0 {
		t.Fatal("room corners still quiet square floor")
	}
	if c.distance(512, 336) < 60 {
		t.Fatal("centre crowded by edge treatment")
	}
	minDepth, maxDepth := 1000, 0
	for x := 320; x < 704; x += 8 {
		depth := 0
		for c.distance(x, 192+depth) < 0 && depth < 128 {
			depth++
		}
		minDepth, maxDepth = min(minDepth, depth), max(maxDepth, depth)
	}
	if maxDepth-minDepth < 24 {
		t.Fatal("edge variation is still just a few pixels")
	}
	plants := c.plants(v)
	if len(plants) < 100 {
		t.Fatal("disconnected thin trim instead of broad planted banks")
	}
	for _, p := range plants {
		if !p.groundCover || (p.role != "bush" && p.role != "verge") || p.rect.Dy() > 28 {
			t.Fatal("bank must contain only low, walkable foliage")
		}
		if c.approach(p.rect) {
			t.Fatal("plant covers entrance approach")
		}
		if p.rect.Overlaps(image.Rect(448, 288, 576, 384)) {
			t.Fatal("quiet centre covered")
		}
		for path := range paths {
			if p.rect.Overlaps(image.Rect(path.X*32+5, path.Y*32+5, path.X*32+27, path.Y*32+27)) {
				t.Fatal("corridor core covered")
			}
		}
	}
}

func TestClearingIgnoresHiddenRoomsAndPaths(t *testing.T) {
	s := domain.NewSessionSeed(21)
	a, b := s.BuildGrid(false), s.BuildGrid(false)
	vis := s.ComputeVisibility(a)
	paths := domain.PathCells(s.Level.Rooms, s.Level.Passages)
	fakePaths := map[domain.Point]bool{}
	for p, v := range paths {
		fakePaths[p] = v
	}
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if !vis.Visible[p] && !vis.Explored[p] {
				b[y][x] = domain.SymRoomFloor
				fakePaths[p] = !fakePaths[p]
			}
		}
	}
	va, vb := newForestView(a, vis), newForestView(b, vis)
	ca, cb := knownClearings(va, paths), knownClearings(vb, fakePaths)
	if !reflect.DeepEqual(ca, cb) {
		t.Fatal("unknown geometry changes visible composition")
	}
	for i := range ca {
		if !reflect.DeepEqual(ca[i].plants(va), cb[i].plants(vb)) {
			t.Fatal("unknown geometry changes plants")
		}
	}
}

func TestClearingProtectsEntrancesInEveryDirection(t *testing.T) {
	for _, d := range forestDirs {
		v, _ := clearingFixture()
		paths := map[domain.Point]bool{}
		// Remove the fixture's existing corridor, then expose only one door.
		for x := 24; x < 30; x++ {
			delete(v.ground, domain.Point{X: x, Y: 10})
		}
		p := domain.Point{X: 16, Y: 10}
		if d.X == 1 {
			p.X = 24
		}
		if d.X == -1 {
			p.X = 7
		}
		if d.Y == 1 {
			p.Y = 15
		}
		if d.Y == -1 {
			p.Y = 5
		}
		v.ground[p] = true
		paths[p] = true
		c := knownClearings(v, paths)[0]
		if len(c.entrances) != 1 {
			t.Fatal("missing entrance")
		}
		e := c.entrances[0]
		for i := 0; i < e.length; i += 4 {
			x, y := e.start.X+e.dir.X*i, e.start.Y+e.dir.Y*i
			if c.distance(x, y) < 20 {
				t.Fatal("planted bank crosses entrance centre")
			}
		}
	}
}

func BenchmarkClearingComposition(b *testing.B) {
	v, paths := clearingFixture()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, c := range knownClearings(v, paths) {
			c.plants(v)
		}
	}
}

func TestClearingBankLightDoesNotFollowRoomRectangle(t *testing.T) {
	v, paths := clearingFixture()
	// The corridor is lit; the adjoining room is remembered. Low vegetation
	// on either side of its border must share the same forest light field.
	v.distance = map[domain.Point]int{}
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			v.distance[domain.Point{X: x, Y: y}] = min(7, max(0, 24-x))
		}
	}
	c := knownClearings(v, paths)[0]
	for _, p := range c.plants(v) {
		if got, want := clearingPlantLight(v, p.rect), float32(.86)*v.light(p.rect); got != want {
			t.Fatal("bank and forest illumination differ")
		}
	}
	rect := image.Rect(23*32+8, 10*32+8, 23*32+24, 10*32+24)
	before := clearingPlantLight(v, rect)
	delete(v.ground, domain.Point{X: 23, Y: 10})
	if clearingPlantLight(v, rect) != before {
		t.Fatal("room membership changes foliage lighting")
	}
	if before < .8 {
		t.Fatal("bank next to lit entrance is darkened as an entire remembered room")
	}
}
