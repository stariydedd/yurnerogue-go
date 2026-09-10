package render

import (
	"image"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// Tagline — слоган под названием в главном меню.
const Tagline = "Every strike may be the last."

// MenuOption — пункт главного меню.
type MenuOption struct {
	Label string
	Key   string
}

// MainMenuOptions — состав главного меню.
var MainMenuOptions = []MenuOption{
	{"PLAY", "new"},
	{"LEADERBOARD", "scoreboard"},
	{"HELP", "help"},
}

// QuitOptions — выбор в диалоге выхода в меню.
var QuitOptions = []MenuOption{
	{"Return to Menu", "menu"},
	{"Cancel", "cancel"},
}

// backdrop — фон экранов меню: сплошной чёрный.
func (r *Renderer) backdrop(screen *ebiten.Image) { screen.Fill(Black) }

// titleWithShadow uses the shared orange menu heading on both layouts.
func (r *Renderer) titleWithShadow(screen *ebiten.Image, s string, y float64) {
	w := TextWidth(s, r.Fonts.Title)
	x := float64(r.Layout.ScreenW)/2 - w/2
	r.Text(screen, s, r.Fonts.Title, x+2, y+2, uiInk)
	r.Text(screen, s, r.Fonts.Title, x, y, uiAccent)
}

// DrawMainMenu — заглавный экран с героем и списком пунктов.
func (r *Renderer) DrawMainMenu(screen *ebiten.Image, selected int, message string) {
	l := r.Layout
	r.drawMenuHeader(screen, message)
	if !l.Touch {
		r.TextCentered(screen, "Click / arrows + Enter", r.Fonts.Small, float64(l.ScreenH)-28, uiMuted)
	}
}

// drawHeroBanner рисует увеличенного игрока по центру экрана.
func (r *Renderer) drawHeroBanner(screen *ebiten.Image, y float64, scale float64) {
	img := r.sprites.Frame("player", r.AnimTick())
	if img == nil {
		return
	}
	w := float64(img.Bounds().Dx()) * scale
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(r.Layout.ScreenW)/2-w/2, y)
	screen.DrawImage(img, op)
}

func NameEntryBounds(l Layout) image.Rectangle {
	w := min(460, l.ScreenW-24)
	return image.Rect((l.ScreenW-w)/2, 420, (l.ScreenW+w)/2, 468)
}

// DrawNameEntry — экран ввода имени перед новой игрой.
func (r *Renderer) DrawNameEntry(screen *ebiten.Image, input string, status ...string) {
	l := r.Layout
	r.backdrop(screen)
	r.titleWithShadow(screen, "YOUR NAME", 170)
	r.drawHeroBanner(screen, 260, 3)

	box := NameEntryBounds(l)
	r.uiSlot(screen, box, true)
	r.Text(screen, input+"_", r.Fonts.Menu, float64(box.Min.X+14), float64(box.Min.Y+14), uiText)

	hint := "Enter: start   Esc: back   (empty = anonymous)"
	if l.Touch {
		hint = "Empty name = anonymous"
	}
	r.TextCentered(screen, hint, r.Fonts.Small, float64(box.Max.Y+26), uiMuted)
	if len(status) > 0 {
		r.drawStatusLines(screen, status[0], float64(box.Max.Y+58), r.Fonts.Small)
	}
}

// helpEntry — строка легенды: спрайт, имя и описание.
type helpEntry struct {
	role, name, desc string
}

var helpEnemies = []helpEntry{
	{"pudge", "Pudge", "Tough and slow. Wanders randomly."},
	{"bloodseeker", "Bloodseeker", "Steals your max HP. Deflects your first strike."},
	{"riki", "Riki", "Blinks around the room, mostly invisible."},
	{"axe", "Axe", "Moves 2 tiles. Rests, counters, never misses."},
	{"skywrath", "Skywrath Mage", "Moves diagonally. Hits may put you to sleep."},
}

var helpItems = []helpEntry{
	{"food", "Tango", "Restores health."},
	{"elixir", "Clarity", "Temporary stat buff for 20 turns."},
	{"scroll", "Scroll", "Permanent stat buff."},
	{"sword", "Weapon", "Equip it; the old one drops nearby."},
	{"portal", "Exit", "Descend deeper. Clear level 21 to win."},
}

// DrawHelp — справка по врагам и предметам.
func (r *Renderer) DrawHelp(screen *ebiten.Image) {
	l := r.Layout
	r.backdrop(screen)
	r.titleWithShadow(screen, "HELP", 24)

	if l.ScreenW < 900 {
		// Узкий экран: одна колонка, компактные строки.
		y := 82.0
		r.Text(screen, "ENEMIES", r.Fonts.UI, 20, y, uiAccent)
		y += 26
		for _, e := range helpEnemies {
			r.helpRowNarrow(screen, e, 20, y, 2)
			y += 50
		}
		y += 8
		r.Text(screen, "ITEMS", r.Fonts.UI, 20, y, uiAccent)
		y += 26
		for _, e := range helpItems {
			r.helpRowNarrow(screen, e, 20, y, 1)
			y += 38
		}
	} else {
		const leftX, rightX, topY = 90.0, 700.0, 120.0
		r.Text(screen, "ENEMIES", r.Fonts.UI, leftX, topY, uiAccent)
		for i, e := range helpEnemies {
			r.helpRow(screen, e, leftX, topY+36+float64(i)*66)
		}
		r.Text(screen, "ITEMS", r.Fonts.UI, rightX, topY, uiAccent)
		for i, e := range helpItems {
			r.helpRow(screen, e, rightX, topY+36+float64(i)*66)
		}
		r.TextCentered(screen, KeyBinds, r.Fonts.Small, float64(l.ScreenH)-132, uiText)
	}

	hint := "Press any key to return..."
	if l.Touch {
		return
	}
	r.TextCentered(screen, hint, r.Fonts.Small, float64(l.ScreenH)-108, uiMuted)
}

// helpRow — строка легенды на широком экране.
func (r *Renderer) helpRow(screen *ebiten.Image, e helpEntry, x, y float64) {
	const slot = 52.0
	r.drawFitted(screen, e.role, x, y, slot)
	r.Text(screen, e.name, r.Fonts.UI, x+slot+14, y+6, uiText)
	r.Text(screen, e.desc, r.Fonts.Small, x+slot+14, y+30, uiMuted)
}

// helpRowNarrow — строка легенды на узком экране с переносом описания.
func (r *Renderer) helpRowNarrow(screen *ebiten.Image, e helpEntry, x, y float64, descLines int) {
	const slot = 36.0
	r.drawFitted(screen, e.role, x, y, slot)
	tx := x + slot + 12
	r.Text(screen, e.name, r.Fonts.UI, tx, y, uiText)
	maxChars := (r.Layout.ScreenW - int(tx) - 8) / 10
	for i, line := range wrapText(e.desc, maxChars) {
		if i >= descLines {
			break
		}
		r.Text(screen, line, r.Fonts.Small, tx, y+18+float64(i)*13, uiMuted)
	}
}

// drawFitted вписывает спрайт роли в квадратный слот.
func (r *Renderer) drawFitted(screen *ebiten.Image, role string, x, y, slot float64) {
	img := r.sprites.Frame(role, r.AnimTick())
	if img == nil {
		return
	}
	w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	scale := 1.0
	if w > slot || h > slot {
		scale = min(slot/w, slot/h)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x+slot/2-w*scale/2, y+slot/2-h*scale/2)
	screen.DrawImage(img, op)
}

// LeaderboardRecord — строка таблицы рекордов.
type LeaderboardRecord struct {
	PlayerName    string
	Treasures     int
	Level         int
	EnemiesKilled int
	FoodUsed      int
	ElixirsUsed   int
	ScrollsRead   int
	AttacksMade   int
	HitsTaken     int
	TilesMoved    int
}

// DrawLeaderboard — таблица рекордов. records == nil означает «ещё грузится».
func (r *Renderer) DrawLeaderboard(screen *ebiten.Image, records []LeaderboardRecord, loading bool, source string) {
	l := r.Layout
	r.backdrop(screen)
	r.titleWithShadow(screen, "HIGH SCORES", 40)
	if source != "" {
		r.TextCentered(screen, source, r.Fonts.Small, 110, uiMuted)
	}

	narrow := l.ScreenW < 900
	switch {
	case loading:
		r.TextCentered(screen, "Loading global leaderboard...", r.Fonts.UI, 170, uiText)
	case len(records) == 0:
		r.TextCentered(screen, "No records yet.", r.Fonts.UI, 170, uiText)
	default:
		// Таблица центрируется целиком, а строки рисуются от общего левого
		// края. Если центрировать каждую строку отдельно, любая разница в
		// ширине (незнакомый шрифту символ в имени) сдвигает всю строку и
		// колонки перестают совпадать.
		header := leaderboardHeader(narrow)
		x := float64(l.ScreenW)/2 - TextWidth(header, r.Fonts.Small)/2
		r.Text(screen, header, r.Fonts.Small, x, 150, uiMuted)
		for i, rec := range records {
			clr := uiText
			if i == 0 {
				clr = uiAccent
			}
			r.Text(screen, leaderboardRow(i+1, rec, narrow), r.Fonts.Small, x, 180+float64(i)*28, clr)
		}
	}

	hint := "Press any key to return..."
	if l.Touch {
		return
	}
	r.TextCentered(screen, hint, r.Fonts.Small, float64(l.ScreenH)-108, uiMuted)
}

func leaderboardHeader(narrow bool) string {
	if narrow {
		return pad("#", 3) + " " + padRight("NAME", 16) + " " + pad("GOLD", 6) + " " + pad("LVL", 4)
	}
	return pad("#", 3) + " " + padRight("NAME", 16) + " " + pad("GOLD", 6) + " " + pad("LVL", 4) + " " +
		pad("KILLS", 5) + " " + pad("FOOD", 4) + " " + pad("ELIX", 4) + " " + pad("SCRL", 4) + " " +
		pad("ATK", 5) + " " + pad("HIT", 5) + " " + pad("MOVE", 6)
}

func leaderboardRow(place int, rec LeaderboardRecord, narrow bool) string {
	row := pad(strconv.Itoa(place), 3) + " " + padRight(playerLabel(rec.PlayerName), 16) + " " +
		pad(strconv.Itoa(rec.Treasures), 6) + " " + pad(strconv.Itoa(rec.Level), 4)
	if narrow {
		return row
	}
	return row + " " +
		pad(strconv.Itoa(rec.EnemiesKilled), 5) + " " + pad(strconv.Itoa(rec.FoodUsed), 4) + " " +
		pad(strconv.Itoa(rec.ElixirsUsed), 4) + " " + pad(strconv.Itoa(rec.ScrollsRead), 4) + " " +
		pad(strconv.Itoa(rec.AttacksMade), 5) + " " + pad(strconv.Itoa(rec.HitsTaken), 5) + " " +
		pad(strconv.Itoa(rec.TilesMoved), 6)
}

// playerLabel готовит имя к выводу в колонку фиксированной ширины.
//
// Имя приходит с сервера и может содержать что угодно: битые байты от старых
// версий ввода, переводы строк, символы шире одной клетки. Всё это ломает
// таблицу, поэтому строка чистится до печатных символов и режется по рунам.
func playerLabel(name string) string {
	name = strings.ToValidUTF8(name, "")
	name = strings.Map(func(r rune) rune {
		if r < ' ' || r == 0x7f {
			return -1
		}
		return r
	}, name)
	if utf8.RuneCountInString(name) > 16 {
		name = string([]rune(name)[:16])
	}
	if name == "" {
		name = "anonymous"
	}
	return name
}

// pad и padRight выравнивают колонки моноширинного шрифта. Ширина считается
// в рунах, а не в байтах: в кириллице символ занимает два байта, и по len()
// имя из восьми букв уже выглядело дополненным до шестнадцати — колонки такой
// строки уезжали относительно остальных.
func pad(s string, width int) string {
	return strings.Repeat(" ", padWidth(s, width)) + s
}

func padRight(s string, width int) string {
	return s + strings.Repeat(" ", padWidth(s, width))
}

func padWidth(s string, width int) int {
	if n := width - utf8.RuneCountInString(s); n > 0 {
		return n
	}
	return 0
}

// DrawEndScreen — экран смерти или победы.
func (r *Renderer) DrawEndScreen(screen *ebiten.Image, title, submitStatus string, customHint ...string) {
	l := r.Layout
	r.backdrop(screen)
	r.titleWithShadow(screen, title, float64(l.ScreenH)/2-80)

	if submitStatus != "" {
		face := r.Fonts.UI
		if l.ScreenW < 900 {
			face = r.Fonts.Small
		}
		r.drawStatusLines(screen, submitStatus, float64(l.ScreenH)/2+20, face)
	}
	hint := "Enter: menu"
	if l.Touch {
		hint = ""
	}
	if len(customHint) > 0 {
		hint = customHint[0]
	}
	r.TextCentered(screen, hint, r.Fonts.Small, float64(l.ScreenH)/2+60, uiMuted)
}

func (r *Renderer) drawStatusLines(screen *ebiten.Image, status string, y float64, face text.Face) {
	for i, line := range r.statusLines(status, face) {
		r.TextCentered(screen, line, face, y+float64(i)*18, uiAccent)
	}
}

func (r *Renderer) statusLines(status string, face text.Face) []string {
	maxChars := max(1, int(float64(r.Layout.ScreenW-40)/TextWidth("M", face)))
	return wrapText(status, maxChars)
}

// DrawItemMenu — список предметов поверх нижней части поля.
func (r *Renderer) DrawItemMenu(screen *ebiten.Image, items []*domain.Item, allowBareHands bool, selected int) {
	l := r.Layout

	var lines []string
	if allowBareHands {
		lines = append(lines, domain.BaseWeaponName)
	}
	for _, it := range items {
		lines = append(lines, it.Name+it.StatLabel())
	}

	rowH := 22.0
	face := r.Fonts.UI
	cancel := "ESC: cancel"
	if l.Touch {
		rowH, face, cancel = 34, r.Fonts.Compact, "MENU: cancel"
	}

	boxH := 10 + float64(len(lines)+1)*rowH
	boxY := float64(l.GridH) - boxH
	vector.DrawFilledRect(screen, 0, float32(boxY), float32(l.ScreenW), float32(boxH), menuBG, false)
	vector.StrokeRect(screen, 0, float32(boxY), float32(l.ScreenW), float32(boxH), 1, menuBorder, false)

	for i, line := range lines {
		prefix, clr := "  ", uiText
		if i == selected {
			prefix, clr = "> ", uiAccent
		}
		// На тач-экране цифровые префиксы бессмысленны — клавиатуры нет.
		// На десктопе номер 0 возвращает базовый Quelling Blade.
		if !l.Touch {
			number := i
			if !allowBareHands {
				number = i + 1
			}
			line = strconv.Itoa(number) + ": " + line
		}
		r.Text(screen, prefix+line, face, 10, boxY+6+float64(i)*rowH, clr)
	}
	r.Text(screen, "  "+cancel, face, 10, boxY+6+float64(len(lines))*rowH, uiMuted)
}

// DrawQuitDialog — подтверждение выхода в главное меню.
func (r *Renderer) DrawQuitDialog(screen *ebiten.Image, selected int) {
	r.dimScreen(screen)
	r.dialogBox(screen, "Return to menu? Run will be lost.", QuitOptions, selected)
}

// dimScreen затемняет кадр под диалогом.
func (r *Renderer) dimScreen(screen *ebiten.Image) {
	l := r.Layout
	vector.DrawFilledRect(screen, 0, 0, float32(l.ScreenW), float32(l.ScreenH), overlayDim, false)
}

// dialogBox рисует окно с заголовком и списком опций.
func (r *Renderer) dialogBox(screen *ebiten.Image, title string, options []MenuOption, selected int) {
	l := r.Layout
	boxW := float64(min(520, l.ScreenW-16))
	boxH := 60 + float64(len(options))*40
	boxX := float64(l.ScreenW)/2 - boxW/2
	boxY := float64(l.ScreenH)/2 - boxH/2

	vector.DrawFilledRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), uiRecess, false)
	vector.StrokeRect(screen, float32(boxX), float32(boxY), float32(boxW), float32(boxH), 2, uiAccent, false)

	r.TextCentered(screen, title, r.Fonts.UI, boxY+16, uiText)
	for i, opt := range options {
		clr := uiText
		if i == selected {
			clr = uiAccent
		}
		r.TextCentered(screen, opt.Label, r.Fonts.UI, boxY+56+float64(i)*40, clr)
	}
}
