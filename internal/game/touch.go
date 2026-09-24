package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// Общий темп удержания клавиатуры и крестовины, в тиках при 60 TPS.
const (
	repeatDelay = 16 // ~260 мс до первого повтора
	// ~200 мс между повторами: шаг при удержании анимируется ровно столько же,
	// поэтому шаги стыкуются в ровную ходьбу без рывков.
	repeatInterval = render.HeldMoveTicks
)

// touchInput переводит касания экранных кнопок в клавиши: игровая логика
// про тач не знает, она видит те же нажатия, что и с клавиатуры.
type touchInput struct {
	controls *render.Controls

	// pressed — какой контрол держит каждый палец (или мышь с id -1).
	pressed map[ebiten.TouchID]string
	// repeatControl и repeatAt — автоповтор зажатой крестовины.
	repeatControl string
	repeatAt      int
	// repeated — последний контрол update пришёл из автоповтора.
	repeated bool
	ticks    int
}

func newTouchInput(c *render.Controls) *touchInput {
	return &touchInput{controls: c, pressed: map[ebiten.TouchID]string{}}
}

func (t *touchInput) reset() {
	clear(t.pressed)
	t.repeatControl = ""
	t.repeatAt = 0
}

// mouseID — псевдопалец для мыши: так тач-раскладку можно щёлкать на десктопе.
const mouseID ebiten.TouchID = -1

// update обрабатывает касания и возвращает контролы, которые надо «нажать».
func (t *touchInput) update(g *Game) []string {
	t.ticks++
	if !ebiten.IsFocused() {
		t.reset()
		return nil
	}
	if _, menu := menuPage(g.state); menu {
		t.reset()
		// Process only one pointer: no second finger or synthetic mouse click
		// may activate another screen during the same tick.
		if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
			g.handleMenuPointer(g.toLogical(ebiten.TouchPosition(ids[0])))
		} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.handleMenuPointer(g.toLogical(ebiten.CursorPosition()))
		}
		return nil
	}
	var fired []string

	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := g.toLogical(ebiten.TouchPosition(id))
		fired = t.press(id, x, y, g.state, fired)
	}
	activeTouches := ebiten.AppendTouchIDs(nil)
	for id := range t.pressed {
		if id == mouseID {
			continue
		}
		active := false
		for _, current := range activeTouches {
			active = active || current == id
		}
		if !active {
			t.release(id)
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := g.toLogical(ebiten.CursorPosition())
		fired = t.press(mouseID, x, y, g.state, fired)
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		t.release(mouseID)
	}

	return append(fired, t.repeatFired()...)
}

// repeatFired возвращает зажатое направление, когда подошло время повтора:
// сперва выдерживается пауза repeatDelay, дальше шаги идут через
// repeatInterval. Вынесено из update, чтобы проверяться без ввода Ebitengine.
func (t *touchInput) repeatFired() []string {
	t.repeated = false
	if t.repeatControl == "" || t.ticks < t.repeatAt {
		return nil
	}
	t.repeated = true
	t.repeatAt = t.ticks + repeatInterval
	return []string{t.repeatControl}
}

// press запоминает палец и возвращает сработавший контрол.
func (t *touchInput) press(id ebiten.TouchID, x, y int, state State, fired []string) []string {
	if _, menu := menuPage(state); menu {
		return fired
	}
	control := t.controls.ControlAt(x, y)
	if control == "" {
		return fired
	}
	t.pressed[id] = control
	if render.DPadControls[control] {
		t.repeatControl = control
		t.repeatAt = t.ticks + repeatDelay
	}
	return append(fired, control)
}

// release снимает палец и гасит автоповтор, если отпустили крестовину.
func (t *touchInput) release(id ebiten.TouchID) {
	control, ok := t.pressed[id]
	if !ok {
		return
	}
	delete(t.pressed, id)
	if control != t.repeatControl {
		return
	}
	// Автоповтор гаснет, только если это направление не держит другой палец.
	for _, held := range t.pressed {
		if held == control {
			return
		}
	}
	t.repeatControl = ""
}

// keyForControl переводит контрол в клавишу с учётом текущего экрана.
func keyForControl(control string, state State) ebiten.Key {
	switch control {
	case render.CtrlUp:
		return ebiten.KeyUp
	case render.CtrlDown:
		return ebiten.KeyDown
	case render.CtrlLeft:
		return ebiten.KeyLeft
	case render.CtrlRight:
		return ebiten.KeyRight
	case render.CtrlRun:
		if !runControlVisible(state) {
			return ebiten.KeyMax
		}
		return ebiten.KeyR
	case render.CtrlStrike:
		if state != StatePlaying {
			return ebiten.KeyMax
		}
		return ebiten.KeyF
	case render.CtrlFood:
		return ebiten.KeyC
	case render.CtrlElixir:
		return ebiten.KeyX
	case render.CtrlGuard:
		if state != StatePlaying {
			return ebiten.KeyMax
		}
		return ebiten.KeyE
	case render.CtrlWait:
		if state != StatePlaying {
			return ebiten.KeyMax
		}
		return ebiten.KeyZ
	case render.CtrlSelect:
		// В игре Enter не нужен — кнопка открывает справку.
		if state == StatePlaying {
			return ebiten.KeyF1
		}
		return ebiten.KeyEnter
	case render.CtrlMenu:
		// В диалогах MENU означает «отмена», иначе — выход в меню.
		switch state {
		case StateItemMenu, StateQuitDialog, StateNameEntry:
			return ebiten.KeyEscape
		}
		return ebiten.KeyQ
	}
	return ebiten.KeyMax
}

func runControlVisible(state State) bool {
	return state == StatePlaying
}

// selectLabel — подпись контекстной кнопки для текущего экрана.
func selectLabel(state State) render.SelectLabel {
	switch state {
	case StatePlaying:
		return render.SelectHelp
	case StateItemMenu:
		return render.SelectUse
	}
	return render.SelectConfirm
}
