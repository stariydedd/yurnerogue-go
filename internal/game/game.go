// Package game связывает домен и рендер: конечный автомат экранов и ввод.
package game

import (
	"runtime"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
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
	StateStarting
	StatePauseMenu
	StateWelcome
	// StateGlossary is the second help page: enemies and items.
	StateGlossary
)

// MaxNameLength — предел длины имени для лидерборда.
const MaxNameLength = 16

// Game — конечный автомат игры, он же ebiten.Game.
type Game struct {
	audio    *sound.Engine
	renderer *render.Renderer
	session  *domain.Session

	state State
	// helpReturn — экран, на который возвращает справка.
	helpReturn        State
	helpScroll        int
	referenceDragging bool
	referenceDragID   ebiten.TouchID
	referenceDragY    int

	menuSelected    int
	menuMessage     string
	quitSelected    int
	pauseSelected   int
	audioDrag       string
	audioDragID     ebiten.TouchID
	runMenuSelected int

	playerName string
	nameInput  string

	// pendingRun — нажата F, ждём направление.
	pendingRun bool

	itemMenuType     domain.ItemType
	itemMenuItems    []*domain.Item
	itemMenuSelected int

	touch    *touchInput
	controls *render.Controls
	keyboard keyboardInput

	// surface is resized with the desktop viewport, independently of game state.
	surface    *ebiten.Image
	outW, outH int

	leaderboard        []render.LeaderboardRecord
	leaderboardLoading bool
	leaderboardSource  string
	submitStatus       string
	topResults         chan topResult
	submitResults      chan error
	runTicket          string
	startResults       chan startResult
	startReturn        State
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

// Layout uses window coordinates for input; desktop content follows its aspect
// ratio while touch keeps its existing logical surface and scaling.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		return max(1, g.outW), max(1, g.outH)
	}
	g.outW, g.outH = outsideWidth, outsideHeight
	if !g.renderer.Layout.Touch {
		layout := render.DesktopLayoutForSize(outsideWidth, outsideHeight)
		layout.Language = g.renderer.Layout.Language
		if layout != g.renderer.Layout {
			g.renderer.Layout = layout
			g.audioDrag = ""
			g.referenceDragging = false
			if page, ok := menuPage(g.state); ok && render.IsHelpPage(page) {
				g.helpScroll = min(g.helpScroll, render.HelpScrollLimit(layout, page))
			}
		}
	}
	return outsideWidth, outsideHeight
}

// State возвращает текущий экран (нужно панели экранных кнопок).
func (g *Game) State() State { return g.state }

// Update обрабатывает ввод; вызывается Ebitengine 60 раз в секунду.
func (g *Game) Update() error {
	before := g.state
	defer func() { g.updateAudio(before) }()
	defer func() {
		if g.state != before {
			g.keyboard.reset()
			if g.touch != nil {
				g.touch.reset()
			}
		}
	}()
	g.pollNetwork()
	browserNameEntry := g.syncBrowserNameEntry()
	defer g.syncBrowserNameEntry()
	g.renderer.Tick()
	if !browserNameEntry {
		g.handleKeyboard(inpututil.AppendJustPressedKeys(nil), ebiten.IsKeyPressed, ebiten.IsFocused())
	} else {
		g.keyboard.reset()
	}
	// Касания экранных кнопок приходят сюда же, переведённые в клавиши.
	if g.updateReferenceScroll() {
		// A scroll gesture cannot activate BACK or another control.
	} else if g.updateAudioSlider() {
		// A captured slider pointer cannot activate another control.
	} else if g.touch != nil {
		for _, control := range g.touch.update(g) {
			if key := keyForControl(control, g.state); key != ebiten.KeyMax {
				g.HandleKey(key)
			}
		}
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := g.toLogical(ebiten.CursorPosition())
		if _, menu := menuPage(g.state); menu {
			g.handleMenuPointer(x, y)
		} else {
			g.handleHUDPointer(x, y)
		}
	}
	if g.state == StateNameEntry && !browserNameEntry {
		g.appendTypedRunes()
	}
	return nil
}

// Desktop slots use the same command path as the keyboard, including replay recording.
func (g *Game) handleHUDPointer(x, y int) {
	if g.state != StatePlaying {
		return
	}
	control := render.HUDControlAt(g.renderer.Layout, x, y)
	if key := keyForControl(control, g.state); key != ebiten.KeyMax {
		g.HandleKey(key)
	}
}

// HandleKey — единая точка входа для клавиш: сюда же приходят нажатия
// экранных кнопок, транслированные в клавиши.
func (g *Game) HandleKey(key ebiten.Key) {
	// Browsers already own F11; do not race their native fullscreen shortcut.
	if runtime.GOOS != "js" && key == ebiten.KeyF11 && g.renderer != nil && !g.renderer.Layout.Touch {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return
	}
	switch g.state {
	case StateStarting:
		if key == ebiten.KeyEscape || key == ebiten.KeyQ {
			g.returnToMenu()
		}
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
	case StatePauseMenu:
		g.handlePauseMenu(key)
	case StateLeaderboard:
		if g.handleReferenceKey(key) {
			return
		}
		switch key {
		case ebiten.KeyEscape, ebiten.KeyQ, ebiten.KeyEnter, ebiten.KeyNumpadEnter:
			g.topResults = nil
			g.state = StateMainMenu
		}
	case StateHelp, StateGlossary:
		if g.handleReferenceKey(key) {
			return
		}
		switch key {
		case ebiten.KeyEscape, ebiten.KeyQ, ebiten.KeyEnter, ebiten.KeyNumpadEnter:
			g.state = g.helpReturn
		case ebiten.KeyLeft, ebiten.KeyA, ebiten.KeyRight, ebiten.KeyD:
			g.switchHelpPage()
		}
	case StateWelcome, StateDeath, StateWin:
		g.handleRunMenu(key)
	}
}

func (g *Game) handleMainMenu(key ebiten.Key) {
	if key == ebiten.KeyM || key == ebiten.KeyV {
		g.audio.Adjust(key == ebiten.KeyM)
		g.audio.Play(sound.Click)
		return
	}
	if g.menuMessage != "" {
		g.menuMessage = ""
		return
	}
	switch key {
	case ebiten.KeyUp, ebiten.KeyW:
		g.menuSelected = (g.menuSelected - 1 + len(render.MainMenuOptions)) % len(render.MainMenuOptions)
		g.audio.Play(sound.Click)
	case ebiten.KeyDown, ebiten.KeyS:
		g.menuSelected = (g.menuSelected + 1) % len(render.MainMenuOptions)
		g.audio.Play(sound.Click)
	case ebiten.KeyLeft, ebiten.KeyA, ebiten.KeyRight, ebiten.KeyD:
		g.adjustAudioSlider(render.MainMenuOptions[g.menuSelected].Key, key)
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		switch render.MainMenuOptions[g.menuSelected].Key {
		case "new":
			g.openWelcome()
		case "scoreboard":
			g.openLeaderboard()
		case "help":
			g.helpScroll = 0
			g.helpReturn = StateMainMenu
			g.state = StateHelp
		case "music", "effects":
			g.audio.Adjust(render.MainMenuOptions[g.menuSelected].Key == "music")
		}
	}
}

func (g *Game) handleNameEntry(key ebiten.Key) {
	switch key {
	case ebiten.KeyEscape:
		g.state = StateMainMenu
	case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
		g.playerName = g.nameInput
		g.requestRankedGame()
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
	s.Message = ""
	if key == ebiten.KeyQ {
		s.Player.StrikeArmed = false
		g.pauseSelected = 0
		g.pendingRun = false
		g.state = StatePauseMenu
		return
	}
	if s.Player.StrikeArmed {
		s.Player.StrikeArmed = false
		if d, ok := directionKeys[key]; ok {
			g.performAction("t" + directionAction(d, false))
			return
		}
		if key == ebiten.KeyF || key == ebiten.KeyEscape {
			return
		}
	}
	if g.pendingRun {
		g.pendingRun = false
		if d, ok := directionKeys[key]; ok {
			g.performAction(directionAction(d, true))
		}
		return
	}
	switch key {
	case ebiten.KeyF:
		if s.Player.Sleeping {
			g.performAction("z")
			return
		}
		if s.Player.StrikeCooldown > 0 {
			s.SetMessage(s.Player.StrikeChargeMessage())
			return
		}
		s.Player.StrikeArmed = true
		s.SetMessage("Critical Strike: choose a direction.")
		return
	case ebiten.KeyE:
		g.performAction("b")
		return
	case ebiten.KeyZ:
		g.performAction("z")
		return
	case ebiten.KeyF1:
		g.helpScroll = 0
		g.helpReturn = StatePlaying
		g.state = StateHelp
		return
	case ebiten.KeyR:
		g.pendingRun = true
		s.SetMessage("Run: press a direction key.")
		return
	}
	itemType := domain.ItemNone
	switch key {
	case ebiten.KeyC:
		itemType = domain.ItemFood
	case ebiten.KeyX:
		itemType = domain.ItemElixir
	}
	if itemType != domain.ItemNone {
		if s.Player.Sleeping {
			g.performAction("z")
		} else {
			g.openItemMenu(itemType)
		}
		return
	}
	if d, ok := directionKeys[key]; ok {
		g.performAction(directionAction(d, false))
	}
}

func directionAction(d domain.Point, run bool) string {
	code := byte('w')
	switch {
	case d.Y > 0:
		code = 's'
	case d.X < 0:
		code = 'a'
	case d.X > 0:
		code = 'd'
	}
	if run {
		code -= 'a' - 'A'
	}
	return string(code)
}

func (g *Game) performAction(action string) {
	g.renderer.SyncMotion(g.session, false)
	before := captureActionAudio(g.session)
	if g.session.ApplyAction(action) == nil {
		animate := g.session.Stats.TilesMoved-before.stats.TilesMoved <= 1
		g.renderer.SyncMotion(g.session, animate)
		if g.renderer != nil {
			g.renderer.ShowCombat(g.session, g.session.CombatEvents)
		}
		for _, cue := range actionCues(before, captureActionAudio(g.session), action) {
			g.audio.Play(cue)
		}
		g.checkGameOver()
	}
}

// openItemMenu открывает выбор предмета нужной категории.
// Ход тратится не здесь, а после выбора.
func (g *Game) openItemMenu(t domain.ItemType) bool {
	items := g.session.Player.ItemsOfType(t)
	if len(items) == 0 {
		g.session.SetMessage("No " + itemTypeName(t) + " in backpack.")
		return false
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
	}
	return "items"
}

func (g *Game) handleItemMenu(key ebiten.Key) {
	rows := len(g.itemMenuItems)

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
		choice = int(key-ebiten.Key0) - 1
	}
	if choice < 0 || choice >= rows {
		return
	}

	code := map[domain.ItemType]byte{domain.ItemFood: 'j', domain.ItemElixir: 'k'}[g.itemMenuType]
	g.state = StatePlaying
	g.performAction(string([]byte{code, byte('0' + choice)}))
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
	g.runTicket = ""
	g.startResults = nil
	g.session = domain.NewSession()
	g.pendingRun = false
	g.submitStatus = ""
	g.submitResults = nil
	g.state = StatePlaying
}

// returnToMenu сбрасывает сессию и возвращается в главное меню.
func (g *Game) returnToMenu() {
	g.runTicket = ""
	g.startResults = nil
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
	if g.state == StateDeath || g.state == StateWin {
		return
	}
	g.state = end
	g.runMenuSelected = 0
	g.submitRun()
}
