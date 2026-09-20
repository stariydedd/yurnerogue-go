package game

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestKeyboardDirectionsRepeatAndRelease(t *testing.T) {
	for key := range directionKeys {
		var input keyboardInput
		down := true
		held := func(k ebiten.Key) bool { return down && k == key }
		if got := input.update([]ebiten.Key{key}, held); got != key {
			t.Fatalf("first press %v: got %v", key, got)
		}
		for tick := 1; tick <= repeatDelay+3*repeatInterval; tick++ {
			want := ebiten.KeyMax
			if tick >= repeatDelay && (tick-repeatDelay)%repeatInterval == 0 {
				want = key
			}
			if got := input.update(nil, held); got != want {
				t.Fatalf("key %v tick %d: got %v, want %v", key, tick, got, want)
			}
		}
		down = false
		for tick := 0; tick < repeatDelay*2; tick++ {
			if input.update(nil, held) != ebiten.KeyMax {
				t.Fatal("movement continued after release")
			}
		}
		if input.update([]ebiten.Key{key}, held) != key {
			t.Fatal("a fresh tap must respond immediately")
		}
	}
}

func TestKeyboardLatestDirectionWinsWithoutDoubleSteps(t *testing.T) {
	var input keyboardInput
	down := map[ebiten.Key]bool{ebiten.KeyW: true, ebiten.KeyD: true}
	held := func(k ebiten.Key) bool { return down[k] }
	if input.update([]ebiten.Key{ebiten.KeyW, ebiten.KeyD}, held) != ebiten.KeyD {
		t.Fatal("latest direction should win")
	}
	for i := 0; i < repeatDelay; i++ {
		key := input.update(nil, held)
		if key != ebiten.KeyMax && key != ebiten.KeyD {
			t.Fatal("older direction also repeated")
		}
	}
	delete(down, ebiten.KeyD)
	if input.update(nil, held) != ebiten.KeyMax {
		t.Fatal("releasing one direction should not make an extra step")
	}
	for i := 1; i <= repeatInterval; i++ {
		got := input.update(nil, held)
		if (i == repeatInterval && got != ebiten.KeyW) || (i < repeatInterval && got != ebiten.KeyMax) {
			t.Fatal("remaining held direction did not resume at the normal pace")
		}
	}
}

func movementTestGame(l render.Layout) *Game {
	g := New(&render.Renderer{Layout: l})
	p := domain.NewPerson()
	p.X, p.Y = 5, 5
	g.session = &domain.Session{Player: p, LevelNum: 1, Level: &domain.Level{
		Rooms: []*domain.Room{{X: 1, Y: 1, W: 30, H: 12}}, Exit: domain.Point{X: 28, Y: 10},
	}}
	g.state = StatePlaying
	return g
}

func TestHeldMovementUsesNormalTurnsAndReplay(t *testing.T) {
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 844)} {
		g := movementTestGame(l)
		held := func(k ebiten.Key) bool { return k == ebiten.KeyD }
		g.handleKeyboard([]ebiten.Key{ebiten.KeyD}, held, true)
		for i := 0; i < repeatDelay+2*repeatInterval; i++ {
			g.handleKeyboard(nil, held, true)
		}
		if g.session.Player.X != 9 || g.session.Turns != 4 || g.session.Actions() != strings.Repeat("d", 4) {
			t.Fatalf("held movement bypassed normal turns: x=%d turns=%d actions=%q", g.session.Player.X, g.session.Turns, g.session.Actions())
		}
	}
}

func TestKeyboardHoldResetsOnPauseAndFocusLoss(t *testing.T) {
	for _, pause := range []bool{false, true} {
		g := movementTestGame(render.DesktopLayout())
		held := func(k ebiten.Key) bool { return k == ebiten.KeyD }
		g.handleKeyboard([]ebiten.Key{ebiten.KeyD}, held, true)
		if pause {
			g.handleKeyboard([]ebiten.Key{ebiten.KeyQ}, held, true)
			g.handleKeyboard([]ebiten.Key{ebiten.KeyEscape}, held, true)
		} else {
			g.handleKeyboard(nil, held, false)
		}
		for i := 0; i < repeatDelay*2; i++ {
			g.handleKeyboard(nil, held, true)
		}
		if g.session.Actions() != "d" {
			t.Fatal("a stale held key resumed movement")
		}
	}
}

func TestKeyboardNonMovementControlsDoNotRepeat(t *testing.T) {
	g := New(&render.Renderer{Layout: render.DesktopLayout()})
	held := func(k ebiten.Key) bool { return k == ebiten.KeyDown }
	g.handleKeyboard([]ebiten.Key{ebiten.KeyDown}, held, true)
	for i := 0; i < repeatDelay*2; i++ {
		g.handleKeyboard(nil, held, true)
	}
	if g.menuSelected != 1 {
		t.Fatal("menu navigation should remain single-press")
	}
}
