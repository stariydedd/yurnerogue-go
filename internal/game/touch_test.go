package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestKeyForControlMapsDirectionsAndItems(t *testing.T) {
	cases := map[string]ebiten.Key{
		render.CtrlUp:     ebiten.KeyUp,
		render.CtrlDown:   ebiten.KeyDown,
		render.CtrlLeft:   ebiten.KeyLeft,
		render.CtrlRight:  ebiten.KeyRight,
		render.CtrlRun:    ebiten.KeyF,
		render.CtrlWeapon: ebiten.KeyH,
		render.CtrlFood:   ebiten.KeyJ,
		render.CtrlElixir: ebiten.KeyK,
		render.CtrlScroll: ebiten.KeyE,
	}
	for control, want := range cases {
		if got := keyForControl(control, StatePlaying); got != want {
			t.Fatalf("%q -> %v, ожидалось %v", control, got, want)
		}
	}
}

func TestSelectIsHelpInGameAndConfirmElsewhere(t *testing.T) {
	// В игре Enter не нужен, поэтому кнопка открывает справку.
	if got := keyForControl(render.CtrlSelect, StatePlaying); got != ebiten.KeyF1 {
		t.Fatalf("SELECT в игре -> %v, ожидалась F1", got)
	}
	for _, state := range []State{StateMainMenu, StateItemMenu, StateDeath, StateNameEntry} {
		if got := keyForControl(render.CtrlSelect, state); got != ebiten.KeyEnter {
			t.Fatalf("SELECT на экране %v -> %v, ожидался Enter", state, got)
		}
	}
}

func TestMenuCancelsInDialogsAndExitsInGame(t *testing.T) {
	// В диалогах MENU означает «отмена», в игре — выход в главное меню.
	for _, state := range []State{StateItemMenu, StateQuitDialog, StateNameEntry} {
		if got := keyForControl(render.CtrlMenu, state); got != ebiten.KeyEscape {
			t.Fatalf("MENU на экране %v -> %v, ожидался Escape", state, got)
		}
	}
	if got := keyForControl(render.CtrlMenu, StatePlaying); got != ebiten.KeyQ {
		t.Fatalf("MENU в игре -> %v, ожидалась Q", got)
	}
}

func TestUnknownControlYieldsNoKey(t *testing.T) {
	if got := keyForControl("", StatePlaying); got != ebiten.KeyMax {
		t.Fatalf("пустой контрол не должен давать клавишу, получено %v", got)
	}
}

func TestSelectLabelFollowsState(t *testing.T) {
	cases := map[State]render.SelectLabel{
		StatePlaying:     render.SelectHelp,
		StateItemMenu:    render.SelectUse,
		StateMainMenu:    render.SelectConfirm,
		StateLeaderboard: render.SelectConfirm,
	}
	for state, want := range cases {
		if got := selectLabel(state); got != want {
			t.Fatalf("подпись на экране %v -> %v, ожидалась %v", state, got, want)
		}
	}
}

func TestDPadHoldAutoRepeats(t *testing.T) {
	// Зажатое направление продолжает шагать: сперва пауза, потом повторы.
	l := render.TouchLayout(390, 844)
	controls := render.NewControls(l)
	ti := newTouchInput(controls)

	ti.pressed[7] = render.CtrlRight
	ti.repeatControl = render.CtrlRight
	ti.repeatAt = repeatDelay

	ti.ticks = repeatDelay - 1
	if fired := ti.repeatFired(); len(fired) != 0 {
		t.Fatal("до истечения паузы повторов быть не должно")
	}
	ti.ticks = repeatDelay
	if fired := ti.repeatFired(); len(fired) != 1 || fired[0] != render.CtrlRight {
		t.Fatalf("ожидался повтор направления, получено %q", fired)
	}

	// Палец отпущен — повторы прекращаются.
	ti.release(7)
	ti.ticks += repeatInterval
	if fired := ti.repeatFired(); len(fired) != 0 {
		t.Fatalf("после отпускания повторов быть не должно, получено %q", fired)
	}
}

func TestReleaseKeepsRepeatWhileAnotherFingerHolds(t *testing.T) {
	l := render.TouchLayout(390, 844)
	ti := newTouchInput(render.NewControls(l))

	ti.pressed[1] = render.CtrlLeft
	ti.pressed[2] = render.CtrlLeft
	ti.repeatControl = render.CtrlLeft

	ti.release(1)
	if ti.repeatControl != render.CtrlLeft {
		t.Fatal("пока направление держит второй палец, повтор должен жить")
	}
	ti.release(2)
	if ti.repeatControl != "" {
		t.Fatal("после отпускания последнего пальца повтор должен погаснуть")
	}
}
