package game

import (
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestStillPlayIsNotRedrawnButAnythingThatMovesIs(t *testing.T) {
	r, err := render.New(render.DesktopLayout())
	if err != nil {
		t.Fatal(err)
	}
	g := New(r)
	g.session = domain.NewSessionSeed(21)
	g.state = StatePlaying
	room := g.session.Level.Rooms[0]
	g.session.Player.X, g.session.Player.Y = room.X+1, room.Y+1
	tick := func() int { // one Update and Draw, returns frames drawn so far
		g.ticks++
		r.Tick()
		surface := g.ensureSurface()
		if !g.reuseFrame(surface) {
			g.drawFrame(surface)
		}
		return g.frames
	}
	for tick() < 1 {
	}
	for r.Animating() { // the ring of chunks ahead of the camera fills first
		tick()
	}
	tick() // the frame after the work settles
	for g.ticks%render.AnimFrameTicks != 0 {
		tick() // start counting right after an animation frame
	}
	tick()
	start := g.frames
	for i := 0; i < render.AnimFrameTicks-2; i++ {
		tick()
	}
	if g.frames != start {
		t.Fatalf("a still frame was redrawn %d times", g.frames-start)
	}
	for i := 0; i < 2*render.AnimFrameTicks; i++ {
		tick()
	}
	if g.frames == start {
		t.Fatal("the idle animation stopped")
	}
	if g.frames-start > 3 {
		t.Fatalf("idle redrew %d frames in %d ticks; only animation frames are needed", g.frames-start, 2*render.AnimFrameTicks)
	}

	// A step: every frame of the slide, one more for the final position.
	g.performAction("d")
	before := g.frames
	for i := 0; i < render.StepTicks; i++ {
		tick()
	}
	if g.frames-before < render.StepTicks-1 {
		t.Fatalf("the step slide drew only %d of %d frames", g.frames-before, render.StepTicks)
	}
	for r.Animating() {
		tick()
	}
	tick()
	settled := g.frames
	for i := 0; i < render.AnimFrameTicks-2; i++ {
		tick()
	}
	if g.frames-settled > 1 {
		t.Fatal("frames kept redrawing after the step settled")
	}

	// A HUD change alone is drawn at once, within the same animation frame.
	surface := g.ensureSurface()
	g.reuseFrame(surface)
	if !g.reuseFrame(surface) {
		t.Fatal("an unchanged frame in the same tick must be reused")
	}
	g.session.SetMessage("You wait.")
	if g.reuseFrame(surface) {
		t.Fatal("a HUD change waited for the next animation frame")
	}
}
