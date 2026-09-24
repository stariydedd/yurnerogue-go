package render

import (
	"image"
	"math"
	"sort"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

const moveTicks = 8 // 133 ms at 60 TPS, independent of simulation turns.

// HeldMoveTicks is one step of held movement: slower than a tap, and linear,
// so consecutive steps join into one steady walk instead of easing in and out
// on every tile. The game repeats a held direction at the same interval.
const HeldMoveTicks = 12 // 200 ms

type actorMotion struct {
	path    []image.Point
	to      image.Point
	started int
	held    bool
}

func (m *actorMotion) position(tick int) image.Point {
	p, _ := m.sample(tick)
	return p
}

func pathLength(path []image.Point) float64 {
	var length float64
	for i := 1; i < len(path); i++ {
		d := path[i].Sub(path[i-1])
		length += math.Hypot(float64(d.X), float64(d.Y))
	}
	return length
}

func (m *actorMotion) sample(tick int) (image.Point, int) {
	if m.held {
		t := min(1., max(0., float64(tick-m.started)/HeldMoveTicks))
		return m.along(t)
	}
	t := min(1., max(0., float64(tick-m.started)/moveTicks))
	// Smoothstep keeps the short slide from starting/stopping abruptly.
	return m.along(t * t * (3 - 2*t))
}

// along returns the point at fraction t of the path.
func (m *actorMotion) along(t float64) (image.Point, int) {
	distance := pathLength(m.path) * t
	for i := 1; i < len(m.path); i++ {
		from, to := m.path[i-1], m.path[i]
		delta := to.Sub(from)
		length := math.Hypot(float64(delta.X), float64(delta.Y))
		if length > 0 && distance < length {
			fraction := distance / length
			return image.Pt(from.X+int(math.Round(float64(delta.X)*fraction)),
				from.Y+int(math.Round(float64(delta.Y)*fraction))), i
		}
		distance -= length
	}
	return m.to, len(m.path)
}

func (m *actorMotion) move(to image.Point, tick int, animate, held bool) {
	if m.to == to {
		return // A blocked step or attack must not restart an existing slide.
	}
	current, next := m.sample(tick)
	delta := to.Sub(m.to)
	if !animate || absMotion(delta.X) > TileSize || absMotion(delta.Y) > TileSize {
		*m = stillMotion(to)
		return
	}
	// Preserve corners when another direction arrives mid-slide. This is a
	// short visual path, never a queue of delayed gameplay commands.
	path := append([]image.Point{current}, m.path[next:]...)
	path = append(path, to)
	straightX, straightY := true, true
	for _, p := range path {
		straightX = straightX && p.X == current.X
		straightY = straightY && p.Y == current.Y
	}
	if straightX || straightY {
		path = []image.Point{current, to} // Reversing direction must not overshoot first.
	}
	if len(path) > 4 || pathLength(path) > 2*TileSize {
		*m = stillMotion(to) // Rapid input must not build up visual lag.
		return
	}
	*m = actorMotion{path: path, to: to, started: tick, held: held}
}

func absMotion(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func stillMotion(p image.Point) actorMotion {
	return actorMotion{to: p}
}

func tilePixels(x, y int) image.Point { return image.Pt(x*TileSize, y*TileSize) }

func (m *actorMotion) pathVisible(vis domain.Visibility) bool {
	for _, p := range m.path {
		if !vis.Visible[domain.Point{X: p.X / TileSize, Y: p.Y / TileSize}] {
			return false
		}
	}
	return true
}

type worldMotion struct {
	session *domain.Session
	held    bool // the next observed step comes from a held direction
	level   *domain.Level
	player  actorMotion
	enemies map[*domain.Opponent]*actorMotion
}

func (m *worldMotion) observe(s *domain.Session, tick int, animate bool) {
	if m.session != s || m.level != s.Level {
		*m = worldMotion{session: s, level: s.Level,
			player: stillMotion(tilePixels(s.Player.X, s.Player.Y)), enemies: map[*domain.Opponent]*actorMotion{}}
		animate = false
	}
	m.player.move(tilePixels(s.Player.X, s.Player.Y), tick, animate, m.held)
	living := make(map[*domain.Opponent]bool)
	for _, op := range s.Opponents() {
		if !op.IsAlive() || !op.IsVisible {
			continue
		}
		living[op] = true
		pos := tilePixels(op.X, op.Y)
		if track := m.enemies[op]; track != nil {
			track.move(pos, tick, animate && (op.Type != domain.Ghost || op.IsChasing), m.held)
		} else {
			track := stillMotion(pos)
			m.enemies[op] = &track
		}
	}
	for op := range m.enemies {
		if !living[op] {
			delete(m.enemies, op)
		}
	}
}

// SetHeldStep marks the next step as part of held movement.
func (r *Renderer) SetHeldStep(held bool) {
	if r != nil {
		r.motion.held = held
	}
}

// SyncMotion brackets ApplyAction. Only the second observation interpolates;
// the simulation has already completed and input never waits for rendering.
func (r *Renderer) SyncMotion(s *domain.Session, animate bool) {
	if r != nil && s != nil {
		r.motion.observe(s, r.tick, animate)
	}
}

func (r *Renderer) actorPosition(s *domain.Session, actor worldActor, vis domain.Visibility) image.Point {
	to := tilePixels(actor.x, actor.y)
	if r.motion.session != s || r.motion.level != s.Level {
		return to
	}
	track := &r.motion.player
	if actor.opponent != nil {
		track = r.motion.enemies[actor.opponent]
	}
	if track == nil || track.to != to {
		return to
	}
	pos := track.position(r.tick)
	// Do not animate a newly revealed enemy out of the fog into the room.
	if actor.opponent != nil && !track.pathVisible(vis) {
		return to
	}
	return pos
}

type movingActor struct {
	worldActor
	position image.Point
}

func (r *Renderer) movingActors(s *domain.Session, vis domain.Visibility) []movingActor {
	var actors []movingActor
	for _, actor := range worldActors(s, vis) {
		actors = append(actors, movingActor{actor, r.actorPosition(s, actor, vis)})
	}
	sort.SliceStable(actors, func(i, j int) bool {
		if actors[i].position.Y == actors[j].position.Y {
			return actors[i].opponent != nil && actors[j].opponent == nil
		}
		return actors[i].position.Y < actors[j].position.Y
	})
	return actors
}

func (r *Renderer) motionCamera(s *domain.Session) (int, int) {
	pos := r.actorPosition(s, worldActor{x: s.Player.X, y: s.Player.Y}, domain.Visibility{})
	l := r.Layout
	return clamp(pos.X+TileSize/2-l.GridW/2, 0, max(0, domain.Cols*TileSize-l.GridW)),
		clamp(pos.Y+TileSize/2-l.GridH/2, 0, max(0, domain.Rows*TileSize-l.GridH))
}
