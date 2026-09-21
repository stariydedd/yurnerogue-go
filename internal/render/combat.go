package render

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

const (
	combatMarkerTicks = 48
	maxCombatMarkers  = 32
	combatLanes       = 4
)

type combatMarker struct {
	event domain.CombatEvent
	born  int
	lane  int
	track *actorMotion
}

type combatMarkers struct {
	session *domain.Session
	level   *domain.Level
	active  []combatMarker
}

func (c *combatMarkers) sync(s *domain.Session) {
	if c.session != s || c.level != s.Level {
		c.active = c.active[:0]
		c.session, c.level = s, s.Level
	}
}

func (c *combatMarkers) expire(tick int) {
	kept := c.active[:0]
	for _, marker := range c.active {
		if tick-marker.born < combatMarkerTicks {
			kept = append(kept, marker)
		}
	}
	c.active = kept
}

// ShowCombat is called once per input action, never once per Draw. Repainting
// cannot restart an effect or duplicate events. A nil renderer is safe in tests.
func (r *Renderer) ShowCombat(s *domain.Session, events []domain.CombatEvent) {
	if r == nil || s == nil {
		return
	}
	c := &r.combat
	c.sync(s)
	c.expire(r.tick)
	for _, event := range events {
		if event.Level != s.LevelNum {
			continue
		}
		var used [combatLanes]bool
		oldest := -1
		for i, marker := range c.active {
			if marker.event.Target == event.Target {
				used[marker.lane] = true
				if oldest == -1 {
					oldest = i
				}
			}
		}
		lane := 0
		for lane < combatLanes && used[lane] {
			lane++
		}
		if lane == combatLanes {
			lane = c.active[oldest].lane
			c.active = append(c.active[:oldest], c.active[oldest+1:]...)
		}
		if len(c.active) == maxCombatMarkers {
			c.active = c.active[1:]
		}
		var track *actorMotion
		if r.motion.session == s && r.motion.level == s.Level {
			if event.TargetPlayer {
				track = &r.motion.player
			} else {
				track = r.motion.enemies[s.OpponentAt(event.Target.X, event.Target.Y)]
			}
		}
		c.active = append(c.active, combatMarker{event: event, born: r.tick, lane: lane, track: track})
	}
}

func combatLabel(event domain.CombatEvent) (string, color.NRGBA) {
	if event.Damage == domain.Miss {
		return "MISS", color.NRGBA{R: 193, G: 210, B: 219, A: 255}
	}
	label := "-" + strconv.Itoa(event.Damage)
	if event.Damage == 0 {
		label = "0"
	}
	if event.MaxHP {
		return label + " MAX HP", color.NRGBA{R: 234, G: 155, B: 255, A: 255}
	}
	if event.TargetPlayer {
		return label, color.NRGBA{R: 255, G: 130, B: 82, A: 255}
	}
	return label, color.NRGBA{R: 255, G: 237, B: 195, A: 255}
}

func (r *Renderer) drawCombat(dst *ebiten.Image, s *domain.Session, vis domain.Visibility, camX, camY int) {
	r.combat.sync(s)
	var labels []image.Rectangle
	for _, marker := range r.combat.active {
		event := marker.event
		if !vis.Visible[event.Target] {
			continue
		}
		age := r.tick - marker.born
		pos := tilePixels(event.Target.X, event.Target.Y)
		if marker.track != nil && marker.track.to == pos && (event.TargetPlayer || marker.track.pathVisible(vis)) {
			pos = marker.track.position(r.tick)
		}
		x := pos.X + TileSize/2 - camX
		feetY := pos.Y + TileSize - camY
		// Do not pin markers for offscreen actors to the edge of the viewport.
		if x < 0 || x >= r.Layout.GridW || feetY < 0 || feetY > r.Layout.GridH+TileSize {
			continue
		}
		label, ink := combatLabel(event)
		if event.Damage == domain.Miss {
			label = r.tr(label)
		} else if event.MaxHP {
			label = strings.TrimSuffix(label, "MAX HP") + r.tr("MAX HP")
		}
		if age > 30 {
			ink.A = uint8(255 * (combatMarkerTicks - age) / (combatMarkerTicks - 30))
		}
		if event.Damage > 0 && age < 8 {
			r.drawImpact(dst, x, feetY-22, age, ink)
		}
		face := r.Fonts.Small
		width := int(TextWidth(label, face))
		// Four vertical lanes keep simultaneous counterattacks readable. Fade
		// and drift use render ticks only, independent of turn count and RNG.
		left := clamp(x-width/2, 2, max(2, r.Layout.GridW-width-2))
		top := clamp(feetY-52-age/3-marker.lane*14, 2, max(2, r.Layout.GridH-14))
		bounds, ok := placeCombatLabel(image.Rect(left, top, left+width, top+12), dst.Bounds(), labels)
		if !ok {
			continue
		}
		labels = append(labels, bounds)
		left, top = bounds.Min.X, bounds.Min.Y
		shadow := color.NRGBA{A: ink.A}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx != 0 || dy != 0 {
					r.Text(dst, label, face, float64(left+dx), float64(top+dy), shadow)
				}
			}
		}
		r.Text(dst, label, face, float64(left), float64(top), ink)
	}
}

// Adjacent actors can have overlapping captions even with separate per-tile
// lanes. Pack labels vertically inside the field, keeping them out of the HUD.
func placeCombatLabel(want, field image.Rectangle, occupied []image.Rectangle) (image.Rectangle, bool) {
	for distance := 0; distance < field.Dy(); distance += 14 {
		for _, direction := range []int{-1, 1} {
			candidate := want.Add(image.Pt(0, distance*direction))
			if !candidate.In(field.Inset(1)) {
				continue
			}
			free := true
			for _, other := range occupied {
				if candidate.Inset(-1).Overlaps(other) {
					free = false
					break
				}
			}
			if free {
				return candidate, true
			}
		}
	}
	return image.Rectangle{}, false
}

func (r *Renderer) drawImpact(dst *ebiten.Image, x, y, age int, ink color.NRGBA) {
	radius := float32(3 + age/2)
	for _, direction := range [][2]float32{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		x1, y1 := float32(x)+direction[0]*radius, float32(y)+direction[1]*radius
		x2, y2 := x1+direction[0]*4, y1+direction[1]*4
		vector.StrokeLine(dst, x1, y1, x2, y2, 4, color.Black, false)
		vector.StrokeLine(dst, x1, y1, x2, y2, 2, ink, false)
	}
}
