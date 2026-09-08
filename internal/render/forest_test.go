package render

import (
	"image"
	"reflect"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestForestDoesNotRevealUnknownGeometry(t *testing.T) {
	s := domain.NewSessionSeed(21)
	a := s.BuildGrid(false)
	vis := s.ComputeVisibility(a)
	b := s.BuildGrid(false)
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if !vis.Visible[p] && !vis.Explored[p] {
				b[y][x] = domain.SymRoomFloor
			}
		}
	}
	va, vb := newForestView(a, vis), newForestView(b, vis)
	if !reflect.DeepEqual(va, vb) {
		t.Fatal("unknown geometry changes forest or lighting")
	}
	viewport := image.Rect(0, 0, 1280, 704)
	if !reflect.DeepEqual(forestProps(va, viewport), forestProps(vb, viewport)) {
		t.Fatal("unknown room changes props")
	}
}

func TestForestKeepsCorridorAndDoorwayClear(t *testing.T) {
	v := forestView{ground: map[domain.Point]bool{}}
	for x := 8; x < 24; x++ {
		v.ground[domain.Point{X: x, Y: 10}] = true
	}
	for y := 6; y < 15; y++ {
		for x := 24; x < 32; x++ {
			v.ground[domain.Point{X: x, Y: y}] = true
		}
	}
	props := forestProps(v, image.Rect(0, 0, 1280, 704))
	if len(props) < 100 {
		t.Fatal("missing forest scenery")
	}
	for _, prop := range props {
		// Flat decals are drawn below the opaque walkable tiles, unlike plants.
		if prop.role == "moss" {
			continue
		}
		for p := range v.ground {
			protected := image.Rect(p.X*32+5, p.Y*32+5, (p.X+1)*32-5, (p.Y+1)*32-5)
			if prop.rect.Overlaps(protected) {
				t.Fatalf("%s covers walkable centre at %v", prop.role, p)
			}
		}
	}
	if v.fits(image.Rect(8*32, 10*32, 9*32, 11*32), 5) {
		t.Fatal("accepted prop over the path")
	}
	if !v.fits(image.Rect(8*32, 9*32, 9*32, 10*32+5), 5) {
		t.Fatal("rejected safe edge fringe")
	}
}

func TestForestStableAcrossFramesAndCameraMovement(t *testing.T) {
	v := forestView{ground: map[domain.Point]bool{{X: 15, Y: 10}: true}}
	a := image.Rect(0, 0, 1280, 704)
	b := a.Add(image.Pt(32, 0))
	props := forestProps(v, a)
	if !reflect.DeepEqual(props, forestProps(v, a)) {
		t.Fatal("scenery flickers between frames")
	}
	common := a.Intersect(b)
	selectProps := func(ps []forestProp) []forestProp {
		var out []forestProp
		for _, p := range ps {
			if p.rect.Overlaps(common) {
				out = append(out, p)
			}
		}
		return out
	}
	if !reflect.DeepEqual(selectProps(props), selectProps(forestProps(v, b))) {
		t.Fatal("camera movement changes world-anchored props")
	}
}

func corridorForestSession() *domain.Session {
	s := domain.NewSessionSeed(21)
	s.Level.Rooms = nil
	s.Level.Items = nil
	s.Level.Passages = []domain.Rect{
		{X: 8, Y: 19, W: 13, H: 3},
		{X: 18, Y: 12, W: 3, H: 10},
	}
	s.Level.Exit = domain.Point{X: 40, Y: 40}
	s.VisitedRooms = map[int]bool{}
	return s
}

func TestForestStableAcrossAdjacentCorridorSteps(t *testing.T) {
	s := corridorForestSession()
	var memory forestMemory
	view := func(p domain.Point) forestView {
		s.Player.X, s.Player.Y = p.X, p.Y
		grid := s.BuildGrid(false)
		return newForestView(grid, memory.reveal(s.Level, grid, s.ComputeVisibility(grid)))
	}
	a, b := domain.Point{X: 19, Y: 20}, domain.Point{X: 19, Y: 19}
	// Discover both views, then walk back and forth around the bend.
	view(a)
	view(b)
	va, vb := view(a), view(b)
	if !reflect.DeepEqual(va.ground, vb.ground) {
		t.Fatal("a corridor disappears after stepping out of sight")
	}
	if reflect.DeepEqual(va.distance, vb.distance) {
		t.Fatal("fixture must change the actual field of view")
	}
	viewport := image.Rect(0, 0, 1280, 900)
	pa, pb := forestProps(va, viewport), forestProps(vb, viewport)
	if !reflect.DeepEqual(pa, pb) {
		t.Fatal("adjacent corridor steps reshuffle scenery")
	}
	if !reflect.DeepEqual(pb, forestProps(view(a), viewport)) {
		t.Fatal("returning around the bend changes scenery")
	}
}

func TestForestDiscoveryOnlyRemovesObstructingProps(t *testing.T) {
	viewport := image.Rect(0, 0, 1280, 900)
	before := forestProps(forestView{}, viewport)
	v := forestView{ground: map[domain.Point]bool{}}
	for y := 10; y <= 20; y++ {
		v.ground[domain.Point{X: 19, Y: y}] = true
	}
	for x := 8; x <= 19; x++ {
		v.ground[domain.Point{X: x, Y: 20}] = true
	}
	solid := func(props []forestProp) map[forestProp]bool {
		result := map[forestProp]bool{}
		for _, p := range props {
			if !p.groundCover && p.role != "verge" {
				result[p] = true
			}
		}
		return result
	}
	a, b := solid(before), solid(forestProps(v, viewport))
	removed := 0
	for p := range a {
		if !b[p] {
			removed++
			if v.fits(p.rect, 5) {
				t.Fatalf("unobstructing %s disappeared at %v", p.role, p.rect)
			}
		}
	}
	for p := range b {
		if !a[p] {
			t.Fatalf("discovery spawned or moved %s at %v", p.role, p.rect)
		}
	}
	if removed == 0 || len(b) == 0 {
		t.Fatal("fixture must remove obstacles and retain surrounding scenery")
	}
}

func TestForestMemoryDoesNotRevealGameplayOrUnknownTerrain(t *testing.T) {
	s := corridorForestSession()
	s.Player.X, s.Player.Y = 19, 20
	grid := s.BuildGrid(false)
	vis := s.ComputeVisibility(grid)
	var memory forestMemory
	memory.reveal(s.Level, grid, vis)
	for p := range memory.ground {
		if !vis.Visible[p] && !vis.Explored[p] {
			t.Fatal("memory contains unseen floor")
		}
	}
	if len(vis.Explored) != 0 {
		t.Fatal("renderer mutated gameplay exploration")
	}
	visible := map[domain.Point]bool{}
	remembered := memory.reveal(s.Level, grid, domain.Visibility{Visible: visible})
	if len(remembered.Explored) == 0 || len(remembered.Visible) != 0 || len(visible) != 0 {
		t.Fatal("remembered terrain must not make items or enemies visible")
	}
	// Changing concealed geometry must not influence the remembered forest.
	other := s.BuildGrid(false)
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			if !vis.Visible[domain.Point{X: x, Y: y}] {
				other[y][x] = domain.SymRoomFloor
			}
		}
	}
	var alternate forestMemory
	got := newForestView(other, alternate.reveal(s.Level, other, vis))
	want := newForestView(grid, memory.reveal(s.Level, grid, vis))
	if !reflect.DeepEqual(got, want) {
		t.Fatal("hidden geometry affects forest memory")
	}
	reset := memory.reveal(&domain.Level{}, grid, domain.Visibility{})
	if len(reset.Explored) != 0 {
		t.Fatal("terrain memory leaked into a new level")
	}
}

func TestForestTreesHaveBreathingRoom(t *testing.T) {
	for _, seed := range []int64{1, 21, 99, 2026} {
		s := domain.NewSessionSeed(seed)
		grid := s.BuildGrid(false)
		v := newForestView(grid, s.ComputeVisibility(grid))
		trees := forestTrees(v)
		if len(trees) < 20 {
			t.Fatalf("seed %d: missing woodland", seed)
		}
		for i, tree := range trees {
			for _, other := range trees[i+1:] {
				crown, trunk := forestTreeSilhouette(tree)
				otherCrown, otherTrunk := forestTreeSilhouette(other)
				if crown.Inset(-6).Overlaps(otherCrown) || crown.Inset(-6).Overlaps(otherTrunk) ||
					trunk.Inset(-6).Overlaps(otherCrown) || trunk.Inset(-6).Overlaps(otherTrunk) {
					t.Fatalf("seed %d: tree silhouettes overlap or lack a 6px gap: %v / %v", seed, tree.rect, other.rect)
				}
			}
		}
	}
}

func TestForestMixesPlantsAndStonesThroughoutWoodland(t *testing.T) {
	// No clearing boundary: bushes and stones must also occur in the forest.
	props := forestProps(forestView{}, image.Rect(0, 0, 1280, 704))
	counts := map[string]int{}
	seenForeground := false
	for _, p := range props {
		if p.groundCover {
			if seenForeground || (p.role != "bush" && p.role != "moss") {
				t.Fatal("understory must be low foliage drawn behind all solid props")
			}
			continue
		}
		seenForeground = true
		counts[p.role]++
	}
	for _, role := range []string{"tree", "bush", "ruins"} {
		if counts[role] < 3 {
			t.Fatalf("missing woodland variety: %s count = %d", role, counts[role])
		}
	}
	for _, tree := range props {
		if tree.role != "tree" {
			continue
		}
		protected := forestSpace{}
		protected.protectTree(tree)
		for _, p := range props {
			if !p.groundCover && (p.role == "bush" || p.role == "ruins") && !protected.free(p.rect) {
				t.Fatalf("%s hides a crown or trunk: %v / %v", p.role, p.rect, tree.rect)
			}
		}
	}
}

func TestForestSilhouetteAllowsInterlockingButProtectsTrunks(t *testing.T) {
	a := forestProp{role: "tree", frame: 3, rect: image.Rect(0, 0, 110, 140)}
	b := forestProp{role: "tree", frame: 0, rect: image.Rect(82, 100, 192, 240)}
	if !a.rect.Overlaps(b.rect) {
		t.Fatal("fixture must overlap transparent sprite margins")
	}
	space := forestSpace{}
	space.protectTree(a)
	crown, trunk := forestTreeSilhouette(b)
	if !space.free(crown.Inset(-6)) || !space.free(trunk.Inset(-6)) {
		t.Fatal("safe staggered trees are unnecessarily separated")
	}
	b.rect = b.rect.Add(image.Pt(-24, 0))
	crown, _ = forestTreeSilhouette(b)
	if space.free(crown.Inset(-6)) {
		t.Fatal("foreground crown can hide the other trunk")
	}
}

func TestForestRootsAlwaysHaveGroundAndSharedLighting(t *testing.T) {
	s := domain.NewSessionSeed(21)
	grid := s.BuildGrid(false)
	v := newForestView(grid, s.ComputeVisibility(grid))
	viewport := image.Rect(0, 0, 1280, 704)
	props := forestProps(v, viewport)
	beds := map[forestProp]bool{}
	seenPlants := false
	for _, p := range props {
		if p.role == "moss" {
			if seenPlants {
				t.Fatal("moss must be in the ground pass before foreground props")
			}
			beds[p] = true
		} else if !p.groundCover {
			seenPlants = true
		} else if len(beds) != 0 {
			t.Fatal("understory must stay below moss, not cover root contact points")
		}
	}
	checked := 0
	for _, p := range props {
		if p.role != "tree" && p.role != "ruins" {
			continue
		}
		bed := forestBed(p)
		if !bed.rect.Overlaps(viewport) {
			continue
		}
		checked++
		if !beds[bed] {
			t.Fatalf("%s has no moss beneath its roots: %v", p.role, p.rect)
		}
		if !forestFoot(p).In(bed.rect) {
			t.Fatal("moss does not support the full footprint")
		}
		if p.lightingBounds() != bed.lightingBounds() {
			t.Fatal("object and ground use different illumination")
		}
		if p.role == "tree" {
			grass := forestRootGrass(p)
			if !grass.rect.In(p.rect) || !v.fits(grass.rect, 5) {
				t.Fatal("root grass intrudes into the path")
			}
		}
	}
	if checked < 10 {
		t.Fatal("missing grounded scenery in test")
	}
}

func TestForestHasDenseSmallDetailsWithoutMovingTrees(t *testing.T) {
	viewport := image.Rect(0, 0, 1280, 704)
	v := forestView{}
	props := forestProps(v, viewport)
	stones, smallPlants, cyan := 0, 0, 0
	trees := map[image.Rectangle]int{}
	for _, p := range props {
		switch {
		case p.role == "tree":
			trees[p.rect] = p.frame
		case p.role == "ruins":
			stones++
		case p.role == "bush" && !p.groundCover && p.rect.Dy() <= 30:
			smallPlants++
			if p.frame == 3 {
				cyan++
			}
		}
	}
	if stones < 25 || smallPlants < 60 || cyan == 0 {
		t.Fatalf("underpopulated forest: %d stones, %d small plants, %d cyan shoots", stones, smallPlants, cyan)
	}
	for _, tree := range forestTrees(v) {
		if !tree.rect.Overlaps(viewport) {
			continue
		}
		if frame, ok := trees[tree.rect]; !ok || frame != tree.frame {
			t.Fatal("small details displaced an existing tree")
		}
	}
	t.Logf("1280x704 woodland: %d stones, %d small plants (%d cyan shoots)", stones, smallPlants, cyan)
}

func TestForestSpaceAcrossBuckets(t *testing.T) {
	s := forestSpace{}
	s.occupy(image.Rect(-70, -20, 150, 130))
	for _, rect := range []image.Rectangle{
		image.Rect(-80, -30, -60, -10), image.Rect(145, 120, 160, 140), image.Rect(64, 64, 65, 65),
	} {
		if s.free(rect) {
			t.Errorf("missed overlap at %v", rect)
		}
	}
	if !s.free(image.Rect(150, 0, 160, 10)) {
		t.Fatal("touching rectangles do not overlap")
	}
}

func BenchmarkForestLayout(b *testing.B) {
	s := domain.NewSessionSeed(21)
	grid := s.BuildGrid(false)
	vis := s.ComputeVisibility(grid)
	for b.Loop() {
		v := newForestView(grid, vis)
		forestProps(v, image.Rect(0, 0, 1280, 704))
	}
}
