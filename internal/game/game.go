// Package game связывает домен и рендер: конечный автомат экранов и ввод.
package game

import (
	"strconv"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// State — экран, на котором сейчас находится игрок.
type State int

const (
	StateMainMenu State = iota
	StateNameEntry
	StatePlaying
	StateItemMenu
	StateQuitDialog
	StateLeaderboard
	StateHelp
	StateDeath
	StateWin
)

// MaxNameLength — предел длины имени для лидерборда.
const MaxNameLength = 16

// Game — конечный автомат игры, он же ebiten.Game.
type Game struct {
	renderer *render.Renderer
	session  *domain.Session

	state State
	// helpReturn — экран, на который возвращает справка.
	helpReturn State

	menuSelected int
	menuMessage  string
	quitSelected int

	playerName string
	nameInput  string

	// pendingRun — нажата F, ждём направление.
	pendingRun bool

	itemMenuType     domain.ItemType
	itemMenuItems    []*domain.Item
	itemMenuBareHand bool
	itemMenuSelected int

	touch    *touchInput
	controls *render.Controls

	// surface — логическая поверхность игры; растягивается на всё окно.
	surface    *ebiten.Image
	outW, outH int

	leaderboard        []render.LeaderboardRecord
	leaderboardLoading bool
	leaderboardSource  string
	submitStatus       string
	topResults         chan topResult
	submitResults      chan error
}

// New создаёт игру с заданным рендерером. На тач-раскладке добавляется
// панель экранных кнопок.
func New(r *render.Renderer) *Game {
	g := &Game{renderer: r, state: StateMainMenu}
	if r.Layout.Touch {
		g.controls = render.NewControls(r.Layout)
		g.touch = newTouchInput(g.controls)
	}
	return g
}

// Layout сообщает Ebitengine размер кадра. Возвращаем физический размер окна
// и растягиваем в него логическую поверхность сами (см. surface.go), иначе
// Ebitengine оставит чёрные поля по краям.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.outW, g.outH = outsideWidth, outsideHeight
	return outsideWidth, outsideHeight
}

// State возвращает текущий экран (нужно панели экранных кнопок).
func (g *Game) State() State { return g.state }

// Update обрабатывает ввод; вызывается Ebitengine 60 раз в секунду.
func (g *Game) Update() error {
	g.pollNetwork()
	g.renderer.Tick()
	for _, key := range inpututil.AppendJustPressedKeys(nil) {
		g.HandleKey(key)
	}
	// Касания экранных кнопок приходят сюда же, переведённые в клавиши.
	if g.touch != nil {
		for _, control := range g.touch.update(g) {
			if key := keyForControl(control, g.state); key != ebiten.KeyMax {
				g.HandleKey(key)
			}
		}
	}
	if g.state == StateNameEntry {
		g.appendTypedRunes()
	}
	return nil
}

// HandleKey — единая точка входа для клавиш: сюда же приходят нажатия
// экранных кнопок, транслированные в клавиши.
func (g *Game) HandleKey(key ebiten.Key) {
	switch g.state {
	case StateMainMenu:
		g.handleMainMenu(key)
	case StateNameEntry:
		g.handleNameEntry(key)
	case StatePlaying:
		g.handlePlaying(key)
	case StateItemMenu:
		g.handleItemMenu(key)
	case StateQuitDialog:
		g.handleQuitDialog(key)
	case StateLeaderboard:
		g.topResults = nil
		g.state = StateMainMenu
	case StateHelp:
		g.state = g.helpReturn
	case StateDeath, StateWin:
		if key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter {
			g.returnToMenu()
		}
	}
}

func (g *Game) handleMainMenu(key ebiten.Key) {
	if g.menuMessage != "" {
		g.menuMessage = ""
		return
	}
	switch key {
	case ebiten.KeyUp, ebiten.KeyW:
		g.menuSelected = (g.menuSelected - 1 + len(render.MainMenuOptions)) % len(render.MainMenuOptions)
	case ebiten.KeyDown, ebiten.KeyS:
		g.menuSelected = (g.menuSelected + 1) % len(render.MainMenuOptions)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		switch render.MainMenuOptions[g.menuSelected].Key {
		case "new":
			g.nameInput = g.playerName
			g.state = StateNameEntry
		case "scoreboard":
			g.openLeaderboard()
		case "help":
			g.helpReturn = StateMainMenu
			g.state = StateHelp
		}
	}
}

func (g *Game) handleNameEntry(key ebiten.Key) {
	switch key {
	case ebiten.KeyEscape:
		g.state = StateMainMenu
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		g.playerName = g.nameInput
		g.startNewGame()
	case ebiten.KeyBackspace:
		// Удаляется символ, а не байт: в кириллице символ занимает два байта,
		// и обрезка по байту оставила бы в имени битую половину руны.
		if n := len(g.nameInput); n > 0 {
			_, size := utf8.DecodeLastRuneInString(g.nameInput)
			g.nameInput = g.nameInput[:n-size]
		}
	}
}

// appendTypedRunes добавляет введённые символы в имя.
func (g *Game) appendTypedRunes() {
	for _, r := range ebiten.AppendInputChars(nil) {
		// Предел тоже в символах, иначе кириллическое имя обрывалось бы вдвое
		// раньше латинского.
		if r >= ' ' && utf8.RuneCountInString(g.nameInput) < MaxNameLength {
			g.nameInput += string(r)
		}
	}
}

// directionKeys — раскладка движения: WASD и стрелки.
var directionKeys = map[ebiten.Key]domain.Point{
	ebiten.KeyW: {X: 0, Y: -1}, ebiten.KeyUp: {X: 0, Y: -1},
	ebiten.KeyS: {X: 0, Y: 1}, ebiten.KeyDown: {X: 0, Y: 1},
	ebiten.KeyA: {X: -1, Y: 0}, ebiten.KeyLeft: {X: -1, Y: 0},
	ebiten.KeyD: {X: 1, Y: 0}, ebiten.KeyRight: {X: 1, Y: 0},
}

func (g *Game) handlePlaying(key ebiten.Key) {
	s := g.session
	sleeping := s.Player.Sleeping
	s.Message = ""

	// Режим бега: следующая клавиша-направление запускает серию ходов,
	// любая другая отменяет. Во сне попытка бега тратит ход, как обычный шаг.
	if g.pendingRun {
		g.pendingRun = false
		if d, ok := directionKeys[key]; ok {
			if sleeping {
				g.resolveTurn(true)
			} else {
				s.Run(d.X, d.Y)
				g.checkGameOver()
			}
		}
		return
	}

	switch key {
	case ebiten.KeyQ:
		g.quitSelected = 0
		g.state = StateQuitDialog
		return
	case ebiten.KeyF1:
		g.helpReturn = StatePlaying
		g.state = StateHelp
		return
	case ebiten.KeyF:
		g.pendingRun = true
		s.SetMessage("Run: press a direction key.")
		return
	}

	acted := false
	switch {
	case key == ebiten.KeyH:
		acted = sleeping || g.openItemMenu(domain.ItemWeapon)
	case key == ebiten.KeyJ:
		acted = sleeping || g.openItemMenu(domain.ItemFood)
	case key == ebiten.KeyK:
		acted = sleeping || g.openItemMenu(domain.ItemElixir)
	case key == ebiten.KeyE:
		acted = sleeping || g.openItemMenu(domain.ItemScroll)
	default:
		d, ok := directionKeys[key]
		if !ok {
			return
		}
		if sleeping {
			acted = true
		} else if d.X != 0 {
			acted = s.MoveX(d.X)
		} else {
			acted = s.MoveY(d.Y)
		}
	}

	if acted {
		g.resolveTurn(sleeping)
	}
}

// resolveTurn завершает ход игрока и проверяет конец игры.
func (g *Game) resolveTurn(sleeping bool) {
	if sleeping {
		g.session.SetMessage("You are asleep!")
	}
	g.session.ResolveTurn()
	g.checkGameOver()
}

// openItemMenu открывает выбор предмета нужной категории.
// Ход тратится не здесь, а после выбора.
func (g *Game) openItemMenu(t domain.ItemType) bool {
	items := g.session.Player.ItemsOfType(t)
	if t == domain.ItemWeapon {
		if len(items) == 0 && g.session.Player.Weapon == nil {
			g.session.SetMessage("No weapons in backpack.")
			return false
		}
		g.itemMenuBareHand = true
	} else {
		if len(items) == 0 {
			g.session.SetMessage("No " + itemTypeName(t) + " in backpack.")
			return false
		}
		g.itemMenuBareHand = false
	}
	g.itemMenuType = t
	g.itemMenuItems = items
	g.itemMenuSelected = 0
	g.state = StateItemMenu
	return false
}

func itemTypeName(t domain.ItemType) string {
	switch t {
	case domain.ItemFood:
		return "food"
	case domain.ItemElixir:
		return "elixirs"
	case domain.ItemScroll:
		return "scrolls"
	}
	return "items"
}

func (g *Game) handleItemMenu(key ebiten.Key) {
	rows := len(g.itemMenuItems)
	if g.itemMenuBareHand {
		rows++
	}

	switch key {
	case ebiten.KeyEscape, ebiten.KeyBackspace:
		g.state = StatePlaying
		return
	case ebiten.KeyUp, ebiten.KeyW:
		g.itemMenuSelected = (g.itemMenuSelected - 1 + rows) % rows
		return
	case ebiten.KeyDown, ebiten.KeyS:
		g.itemMenuSelected = (g.itemMenuSelected + 1) % rows
		return
	}

	choice := -1
	switch {
	case key == ebiten.KeyEnter || key == ebiten.KeyNumpadEnter:
		choice = g.itemMenuSelected
	case key >= ebiten.Key0 && key <= ebiten.Key9:
		digit := int(key - ebiten.Key0)
		if g.itemMenuBareHand {
			choice = digit
		} else {
			choice = digit - 1
		}
	}
	if choice < 0 || choice >= rows {
		return
	}

	g.applyItemChoice(choice)
	g.state = StatePlaying
	g.resolveTurn(false)
}

// applyItemChoice применяет выбранный предмет: экипирует оружие или использует
// расходник, обновляя статистику.
func (g *Game) applyItemChoice(choice int) {
	s := g.session
	p := s.Player

	if g.itemMenuType == domain.ItemWeapon {
		if choice == 0 {
			weapon := p.UnequipWeapon()
			if weapon == nil {
				return
			}
			if p.PickUpItem(weapon) {
				s.SetMessage("You holstered " + weapon.Name + ".")
			} else {
				p.Weapon = weapon
				s.SetMessage("Backpack full! Cannot holster weapon.")
			}
			return
		}
		item := g.itemMenuItems[choice-1]
		old := p.EquipWeapon(item)
		msg := "You equipped " + item.Name + " [+" + strconv.Itoa(item.StrengthEffect) + " STR]."
		if old != nil {
			if s.DropItemNearPlayer(old) {
				msg += " Dropped " + old.Name + "."
			} else {
				// EquipWeapon освободил слот оружия, поэтому прежнее помещается
				// даже в рюкзак, который был полон до смены.
				p.Backpack = append(p.Backpack, old)
				msg += " Stowed " + old.Name + " in backpack."
			}
		}
		s.SetMessage(msg)
		return
	}

	item := g.itemMenuItems[choice]
	if !p.UseItem(item) {
		return
	}
	s.SetMessage("You used " + item.Name + item.StatLabel() + ".")
	switch g.itemMenuType {
	case domain.ItemFood:
		s.Stats.FoodUsed++
	case domain.ItemElixir:
		s.Stats.ElixirsUsed++
	case domain.ItemScroll:
		s.Stats.ScrollsRead++
	}
}

func (g *Game) handleQuitDialog(key ebiten.Key) {
	switch key {
	case ebiten.KeyUp, ebiten.KeyW:
		g.quitSelected = (g.quitSelected - 1 + len(render.QuitOptions)) % len(render.QuitOptions)
	case ebiten.KeyDown, ebiten.KeyS:
		g.quitSelected = (g.quitSelected + 1) % len(render.QuitOptions)
	case ebiten.KeyEscape:
		g.state = StatePlaying
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		if render.QuitOptions[g.quitSelected].Key == "menu" {
			g.returnToMenu()
		} else {
			g.state = StatePlaying
		}
	}
}

// startNewGame начинает забег.
func (g *Game) startNewGame() {
	g.session = domain.NewSession()
	g.pendingRun = false
	g.submitStatus = ""
	g.submitResults = nil
	g.state = StatePlaying
}

// returnToMenu сбрасывает сессию и возвращается в главное меню.
func (g *Game) returnToMenu() {
	g.topResults = nil
	g.submitResults = nil
	g.session = nil
	g.menuSelected = 0
	g.state = StateMainMenu
}

// checkGameOver переводит на экран смерти или победы.
func (g *Game) checkGameOver() {
	switch {
	case !g.session.Player.IsAlive():
		g.finishRun(StateDeath)
	case g.session.Won():
		g.finishRun(StateWin)
	}
}

// finishRun завершает забег и отправляет результат в лидерборд.
func (g *Game) finishRun(end State) {
	g.state = end
	g.submitRun()
}
