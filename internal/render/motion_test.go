package render

import (
	"image"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestMotionInterpolatesAndRetargetsWithoutRestartJump(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	m.move(image.Pt(352, 320), 10, true)
	if m.position(10) != image.Pt(320, 320) || m.position(14) != image.Pt(336, 320) || m.position(18) != m.to {
		t.Fatal("incorrect motion endpoints or midpoint")
	}
	m.move(image.Pt(384, 320), 14, true)
	if m.position(14) != image.Pt(336, 320) || m.position(22) != image.Pt(384, 320) {
		t.Fatal("rapid input jumped to an old tile or failed to settle")
	}
	started := m.started
	m.move(m.to, 16, true)
	if m.started != started {
		t.Fatal("blocked movement restarted interpolation")
	}
}

func TestMotionKeepsCornerAndBoundsVisualBacklog(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	m.move(image.Pt(352, 320), 0, true)
	m.move(image.Pt(352, 288), 4, true)
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
		m.move(m.to.Add(image.Pt(32, 0)), 5, true)
		if len(m.path) > 4 || pathLength(m.path) > 2*TileSize {
			t.Fatal("unbounded visual backlog")
		}
	}
}

func TestMotionReversalDoesNotOvershoot(t *testing.T) {
	m := stillMotion(image.Pt(320, 320))
	m.move(image.Pt(352, 320), 0, true)
	m.move(image.Pt(320, 320), 4, true)
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
		key := forestCacheKey{level: s.Level, player: domain.Point{X: 40, Y: 25}, viewport: image.Rect(100, 100, 100+l.GridW, 100+l.GridH)}
		cached := key
		cached.viewport = cached.viewport.Inset(-2 * TileSize)
		for offset := -32; offset <= 32; offset++ {
			request := key
			request.viewport = request.viewport.Add(image.Pt(offset, -offset))
			if !forestCacheContains(cached, request) {
				t.Fatal("camera interpolation invalidates terrain cache")
			}
		}
		key.player.X++
		if forestCacheContains(cached, key) {
			t.Fatal("new visibility reused stale terrain")
		}
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
