package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"testing"

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

func TestForestCacheCoversEveryPixelOfShortCameraSlide(t *testing.T) {
	s := domain.NewSessionSeed(21)
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 700)} {
		key := forestCacheKey{level: s.Level, visible: 1, viewport: image.Rect(100, 100, 100+l.GridW, 100+l.GridH)}
		cached := key
		cached.viewport = cached.viewport.Inset(-2 * TileSize)
		for offset := -32; offset <= 32; offset++ {
			request := key
			request.viewport = request.viewport.Add(image.Pt(offset, -offset))
			if !forestCacheContains(cached, request) {
				t.Fatal("camera interpolation invalidates terrain cache")
			}
		}
		key.visible++
		if forestCacheContains(cached, key) {
			t.Fatal("new visibility reused stale terrain")
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

func TestForestRebuildSpreadsOverFramesWhileOldTerrainStays(t *testing.T) {
	r, err := New(DesktopLayout())
	if err != nil {
		t.Fatal(err)
	}
	s := domain.NewSessionSeed(21)
	grid := s.BuildGrid(false)
	vis := s.ComputeVisibility(grid)
	paths := r.levelPaths(s.Level)
	dst := ebiten.NewImage(r.Layout.GridW, r.Layout.GridH)
	key := forestCacheKey{level: s.Level, visible: 1, viewport: image.Rect(640, 320, 640+r.Layout.GridW, 320+r.Layout.GridH)}
	frame := func(k forestCacheKey) { r.drawCachedForest(dst, grid, vis, paths, k) }

	frame(key) // nothing to show yet: built at once
	if r.forest == nil || r.forestBuild != nil {
		t.Fatal("first terrain was not built in one frame")
	}
	near := key
	near.viewport = near.viewport.Add(image.Pt(TileSize/2, 0))
	frame(near)
	if r.forestBuild != nil {
		t.Fatal("a short camera slide started a rebuild")
	}
	for name, next := range map[string]forestCacheKey{
		"camera": {level: key.level, visible: key.visible, viewport: key.viewport.Add(image.Pt(TileSize*3/2, 0))},
		"light":  {level: key.level, visible: key.visible + 1, viewport: key.viewport},
	} {
		frame(key)
		for r.forestBuild != nil {
			frame(key)
		}
		old := r.forest
		for i := 1; i < forestStrips; i++ {
			frame(next)
			if r.forest != old || r.forestBuild == nil || r.forestBuild.strip != i {
				t.Fatalf("%s: frame %d did not draw exactly one strip over the old terrain", name, i)
			}
		}
		frame(next)
		if r.forestBuild != nil || !forestCacheContains(r.forest.key, next) {
			t.Fatalf("%s: rebuild did not finish after %d frames", name, forestStrips)
		}
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
		if d := xs[i] - xs[i-1]; d < 2 || d > 3 {
			t.Fatalf("tick %d moved %d px; held walking must keep one speed across tiles", i, d)
		}
	}
	if HeldMoveTicks <= moveTicks {
		t.Fatal("held movement must be slower than a tapped step")
	}
}
