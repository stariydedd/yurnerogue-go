package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"testing"
	"time"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestMotionInterpolatesAndRetargetsWithoutRestartJump(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	m.move(image.Pt(352, 320), 10, true, false)
	if m.position(10) != image.Pt(320, 320) || m.position(14) != image.Pt(336, 320) || m.position(18) != m.to {
		t.Fatal("incorrect motion endpoints or midpoint")
	}
	m.move(image.Pt(384, 320), 14, true, false)
	if m.position(14) != image.Pt(336, 320) || m.position(22) != image.Pt(384, 320) {
		t.Fatal("rapid input jumped to an old tile or failed to settle")
	}
	started := m.started
	m.move(m.to, 16, true, false)
	if m.started != started {
		t.Fatal("blocked movement restarted interpolation")
	}
}

func TestMotionKeepsCornerAndBoundsVisualBacklog(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	m.move(image.Pt(352, 320), 0, true, false)
	m.move(image.Pt(352, 288), 4, true, false)
	if m.position(4) != image.Pt(336, 320) {
		t.Fatal("turn jumped to corner")
	}
	for tick := 4; tick <= 12; tick++ {
		p := m.position(tick)
		if p.Y != 320 && p.X != 352 {
			t.Fatalf("diagonal shortcut through corridor corner: %v", p)
		}
	}
	for i := 0; i < 100; i++ {
		m.move(m.to.Add(image.Pt(32, 0)), 5, true, false)
		if len(m.path) > 4 || pathLength(m.path) > 2*TileSize {
			t.Fatal("unbounded visual backlog")
		}
	}
}

func TestMotionReversalDoesNotOvershoot(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	m.move(image.Pt(352, 320), 0, true, false)
	m.move(image.Pt(320, 320), 4, true, false)
	previous := 336
	for tick := 4; tick <= 12; tick++ {
		p := m.position(tick)
		if p.X > previous || p.X < 320 {
			t.Fatal("reversing direction first continues toward obsolete destination")
		}
		previous = p.X
	}
}

func TestMotionSnapsRunTeleportNewLevelAndNewSession(t *testing.T) {
	r := &Renderer{Layout: TouchLayout(390, 700)}
	s := domain.NewSessionSeed(21)
	r.SyncMotion(s, false)
	s.Player.X++
	r.SyncMotion(s, true)
	s.Player.X += 8
	r.SyncMotion(s, false)
	if r.motion.player.position(r.tick) != tilePixels(s.Player.X, s.Player.Y) {
		t.Fatal("RUN draws a straight line across the map")
	}
	s.Player.X += 10
	r.SyncMotion(s, true)
	if r.motion.player.position(r.tick) != tilePixels(s.Player.X, s.Player.Y) {
		t.Fatal("teleport was interpolated")
	}
	s.UpdateLevel()
	r.SyncMotion(s, true)
	if r.motion.player.position(r.tick) != tilePixels(s.Player.X, s.Player.Y) {
		t.Fatal("new floor slides in from previous floor")
	}
	s = domain.NewSessionSeed(22)
	r.SyncMotion(s, true)
	if r.motion.session != s || r.motion.player.position(r.tick) != tilePixels(s.Player.X, s.Player.Y) {
		t.Fatal("old run motion retained")
	}
}

func TestMotionEnemyIdentityVisibilityAndDepth(t *testing.T) {
	s := domain.NewSessionSeed(21)
	s.Player.X, s.Player.Y = 16, 10
	a, b := domain.NewOpponent(domain.Zombie), domain.NewOpponent(domain.Zombie)
	a.X, a.Y, b.X, b.Y = 17, 11, 18, 9
	s.Level.Rooms = []*domain.Room{{Enemies: []*domain.Opponent{a, b}}}
	r := &Renderer{}
	r.SyncMotion(s, false)
	a.Y--
	b.Y++
	r.SyncMotion(s, true)
	vis := domain.Visibility{Visible: map[domain.Point]bool{}}
	for y := 8; y <= 12; y++ {
		for x := 15; x <= 19; x++ {
			vis.Visible[domain.Point{X: x, Y: y}] = true
		}
	}
	actors := r.movingActors(s, vis)
	if actors[0].opponent != b || actors[2].opponent != a {
		t.Fatal("depth used destination cells or confused same-role enemies")
	}
	delete(vis.Visible, domain.Point{X: 17, Y: 11})
	p := r.actorPosition(s, worldActor{x: a.X, y: a.Y, opponent: a}, vis)
	if p != tilePixels(a.X, a.Y) {
		t.Fatal("enemy animated out of unseen terrain")
	}
	r.tick = 4
	if p = r.actorPosition(s, worldActor{x: a.X, y: a.Y, opponent: a}, vis); p != tilePixels(a.X, a.Y) {
		t.Fatal("newly revealed enemy jumped back halfway through its slide")
	}
	a.Health, b.IsVisible = 0, false
	r.SyncMotion(s, true)
	if len(r.motion.enemies) != 0 || len(r.movingActors(s, vis)) != 1 {
		t.Fatal("dead or invisible actors retained")
	}
}

func TestMotionCameraUsesSamePixelPositionAndClampsEdges(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 700)} {
		r := &Renderer{Layout: l}
		s := domain.NewSessionSeed(21)
		s.Player.X, s.Player.Y = 40, 25
		r.SyncMotion(s, false)
		x0, y0 := r.motionCamera(s)
		s.Player.X++
		r.SyncMotion(s, true)
		r.tick = moveTicks / 2
		x, y := r.motionCamera(s)
		if x != x0+16 || y != y0 {
			t.Fatal("camera jumped ahead of player")
		}
		for _, p := range []domain.Point{{}, {X: domain.Cols - 1, Y: domain.Rows - 1}} {
			s.Player.X, s.Player.Y = p.X, p.Y
			r.SyncMotion(s, false)
			x, y := r.motionCamera(s)
			wantX, wantY := cameraOffset(l, p.X, p.Y)
			if x != wantX || y != wantY {
				t.Fatal("camera moved outside map")
			}
		}
	}
}

func TestForestCacheSurvivesStepsInsideARoom(t *testing.T) {
	s := domain.NewSessionSeed(21)
	room := s.Level.Rooms[0]
	signature := func(x, y int) uint64 {
		s.Player.X, s.Player.Y = x, y
		return visibleSignature(s.ComputeVisibility(s.BuildGrid(false)).Visible)
	}
	here := signature(room.X, room.Y)
	if signature(room.X+room.W-1, room.Y+room.H-1) != here {
		t.Fatal("a step inside the room would redraw the whole forest")
	}
	if other := s.Level.Rooms[1]; signature(other.X, other.Y) == here {
		t.Fatal("another room reused this room's terrain")
	}
}

func TestCombatMarkerTracksMovingTargetUntilImpactTileChanges(t *testing.T) {
	r := &Renderer{}
	s := domain.NewSessionSeed(21)
	r.SyncMotion(s, false)
	s.Player.X++
	r.SyncMotion(s, true)
	e := domain.CombatEvent{Target: domain.Point{X: s.Player.X, Y: s.Player.Y}, Level: s.LevelNum, Damage: 10, TargetPlayer: true}
	r.ShowCombat(s, []domain.CombatEvent{e})
	marker := r.combat.active[0]
	if marker.track != &r.motion.player || marker.track.position(0) == tilePixels(e.Target.X, e.Target.Y) {
		t.Fatal("impact marker jumped ahead of sliding player")
	}
}

func TestHeldStepsWalkAtSteadySpeedWithoutEasing(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	var xs []int
	for step := 0; step < 3; step++ {
		start := step * HeldMoveTicks
		m.move(image.Pt(320+TileSize*(step+1), 320), start, true, true)
		for tick := start; tick < start+HeldMoveTicks; tick++ {
			xs = append(xs, m.position(tick).X)
		}
	}
	for i := 1; i < len(xs); i++ {
		if d := xs[i] - xs[i-1]; d < TileSize/HeldMoveTicks || d > (TileSize+HeldMoveTicks-1)/HeldMoveTicks {
			t.Fatalf("tick %d moved %d px; held walking must keep one speed across tiles", i, d)
		}
	}
	if HeldMoveTicks != moveTicks {
		t.Fatal("held movement must keep the tapped step pace")
	}
}

// chunkFixture is a renderer and a session standing in its first room.
func chunkFixture(t *testing.T) (*Renderer, *domain.Session, func(image.Rectangle) int) {
	r, err := New(DesktopLayout())
	if err != nil {
		t.Fatal(err)
	}
	s := domain.NewSessionSeed(21)
	room := s.Level.Rooms[0]
	s.Player.X, s.Player.Y = room.X, room.Y
	dst := ebiten.NewImage(r.Layout.GridW, r.Layout.GridH)
	none := time.Duration(0) // one stale chunk per frame: deterministic
	r.forest.budget = &none
	// frame draws one frame and returns how many chunks it redrew.
	frame := func(view image.Rectangle) int {
		grid := s.BuildGrid(false)
		vis := s.ComputeVisibility(grid)
		before := r.forest.drawn
		r.drawCachedForest(dst, s.Level, grid, vis, visibleSignature(vis.Visible), r.levelPaths(s.Level), len(s.VisitedRooms), view)
		return r.forest.drawn - before
	}
	return r, s, frame
}

func TestForestChunksRedrawOnlyWhatChanged(t *testing.T) {
	r, s, frame := chunkFixture(t)
	view := image.Rect(600, 300, 600+r.Layout.GridW, 300+r.Layout.GridH)
	lo, hi := chunkRange(view)
	onScreen := (hi.X - lo.X + 1) * (hi.Y - lo.Y + 1)
	if n := frame(view); n < onScreen || n > onScreen+1 {
		t.Fatalf("first frame drew %d chunks, want the %d on screen plus one ahead", n, onScreen)
	}
	for i := 0; i < 40 && frame(view) > 0; i++ {
	}
	if frame(view) != 0 {
		t.Fatal("a settled forest keeps redrawing")
	}
	// A step inside the room and a short slide within the same chunks.
	room := s.Level.Rooms[0]
	s.Player.X = room.X + 1
	if n := frame(view.Add(image.Pt(TileSize, 0))); n != 0 {
		t.Fatalf("a step inside a room redrew %d chunks", n)
	}
	// New light: the chunk under the middle of the view first, then one per
	// frame within the budget.
	s.Player.X, s.Player.Y = s.Level.Rooms[1].X, s.Level.Rooms[1].Y
	total := 0
	middle, _ := chunkRange(image.Rectangle{Min: view.Min.Add(view.Size().Div(2)), Max: view.Min.Add(view.Size().Div(2)).Add(image.Pt(1, 1))})
	for i := 0; i < 40; i++ {
		n := frame(view)
		if i == 0 && n == 1 && r.forest.last != middle {
			t.Fatalf("first redraw was chunk %v, not %v under the middle of the view", r.forest.last, middle)
		}
		if n > 1 {
			t.Fatalf("frame redrew %d chunks already on screen", n)
		}
		total += n
	}
	if total == 0 {
		t.Fatal("changed light never reached the chunks")
	}
}

func TestForestChunkKeyIgnoresChangesFarAway(t *testing.T) {
	var f forestChunks
	f.view = forestView{lights: make([]float32, domain.Cols*domain.Rows), cells: make([]bool, domain.Cols*domain.Rows), seen: make([]bool, domain.Cols*domain.Rows)}
	c := image.Pt(2, 2)
	before := f.key(c)
	f.view.lights[40*domain.Cols+90] = 1 // far corner of the map
	if f.key(c) != before {
		t.Fatal("a far change touched the chunk")
	}
	f.view.lights[(2*chunkTiles+3)*domain.Cols+2*chunkTiles+3] = 1 // inside the chunk
	if f.key(c) == before {
		t.Fatal("a change inside the chunk was missed")
	}
	before = f.key(c)
	f.view.seen[(2*chunkTiles-chunkMargin)*domain.Cols+2*chunkTiles] = true // in the margin
	if f.key(c) == before {
		t.Fatal("a change in the margin was missed")
	}
}

func TestRedrawnChunksFadeInOverTheirOldPicture(t *testing.T) {
	r, s, frame := chunkFixture(t)
	view := image.Rect(600, 300, 600+r.Layout.GridW, 300+r.Layout.GridH)
	for i := 0; i < 40; i++ {
		r.Tick()
		frame(view)
	}
	s.Player.X, s.Player.Y = s.Level.Rooms[1].X, s.Level.Rooms[1].Y
	var faded *forestChunk
	for i := 0; i < 20 && faded == nil; i++ {
		r.Tick()
		frame(view)
		faded = r.forest.chunks[r.forest.last]
		if faded != nil && faded.old == nil {
			faded = nil
		}
	}
	if faded == nil {
		t.Fatal("a redrawn chunk replaced its picture without fading")
	}
	if !r.Animating() {
		t.Fatal("a fading chunk must keep frames coming")
	}
	pool := len(r.forest.pool)
	for i := 0; i <= chunkFadeTicks; i++ {
		r.Tick()
		frame(view)
	}
	if faded.old != nil || len(r.forest.pool) <= pool {
		t.Fatal("the old picture was not released after the fade")
	}
}
