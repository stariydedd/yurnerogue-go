package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestDesktopHUDPointerUsesKeyboardActions(t *testing.T) {
	for _, control := range []string{render.CtrlFood, render.CtrlWeapon, render.CtrlElixir, render.CtrlScroll, render.CtrlMenu, render.CtrlSelect} {
		newGame := func() *Game {
			s := domain.NewSessionSeed(21)
			for _, kind := range []domain.ItemType{domain.ItemFood, domain.ItemElixir, domain.ItemScroll, domain.ItemWeapon} {
				s.Player.PickUpItem(&domain.Item{Type: kind})
			}
			return &Game{renderer: &render.Renderer{Layout: render.DesktopLayout()}, session: s, state: StatePlaying}
		}
		pointer, keyboard := newGame(), newGame()
		box := render.HUDTargets(pointer.renderer.Layout)[control]
		pointer.handleHUDPointer(box.Min.X+box.Dx()/2, box.Min.Y+box.Dy()/2)
		keyboard.HandleKey(keyForControl(control, StatePlaying))
		if pointer.state != keyboard.state || pointer.itemMenuType != keyboard.itemMenuType || len(pointer.itemMenuItems) != len(keyboard.itemMenuItems) {
			t.Fatalf("pointer %s differs from keyboard", control)
		}
		if len(pointer.session.Player.Backpack) != 4 {
			t.Fatal("opening a slot consumed an item")
		}
	}
}

func TestHUDPointerCannotActivateSlotsBehindDialogs(t *testing.T) {
	g := &Game{renderer: &render.Renderer{Layout: render.DesktopLayout()}, state: StateMainMenu}
	for _, state := range []State{StateMainMenu, StateItemMenu, StateQuitDialog, StateHelp, StateDeath, StateStarting} {
		g.state = state
		g.handleHUDPointer(880, g.renderer.Layout.GridH+50)
		if g.state != state {
			t.Fatalf("hidden HUD changed state %v", state)
		}
	}
}

func TestRunHighlightPersistsUntilDirection(t *testing.T) {
	g := &Game{session: domain.NewSessionSeed(21), state: StatePlaying}
	g.HandleKey(ebiten.KeyF)
	if !g.pressedControls()[render.CtrlRun] {
		t.Fatal("armed run is not highlighted")
	}
	g.HandleKey(ebiten.KeyRight)
	if g.pressedControls()[render.CtrlRun] {
		t.Fatal("run highlight survived direction")
	}
	for _, state := range []State{StateMainMenu, StateItemMenu, StateQuitDialog, StateStarting, StateHelp, StateDeath, StateWin} {
		if runControlVisible(state) || keyForControl(render.CtrlRun, state) != ebiten.KeyMax {
			t.Fatalf("run available on screen %v", state)
		}
	}
}
