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
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

// Tagline — слоган под названием в главном меню.
const Tagline = "Every strike may be the last"

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
	{"MUSIC", "music"},
	{"SFX", "effects"},
}

// QuitOptions — выбор в диалоге выхода в меню.
var QuitOptions = []MenuOption{
	{"MAIN MENU", "menu"},
	{"CANCEL", "cancel"},
}

// backdrop — фон экранов меню: сплошной чёрный.
func (r *Renderer) backdrop(screen *ebiten.Image) { screen.Fill(Black) }

// titleWithShadow uses the shared orange menu heading on both layouts.
func (r *Renderer) titleWithShadow(screen *ebiten.Image, s string, y float64) {
	s = r.tr(s)
	face := r.Fonts.Title
	if TextWidth(s, face) > float64(r.Layout.ScreenW-32) {
		face = r.Fonts.Menu
	}
	w := TextWidth(s, face)
	x := float64(r.Layout.ScreenW)/2 - w/2
	r.Text(screen, s, face, x+2, y+2, uiInk)
	r.Text(screen, s, face, x, y, uiAccent)
}

// DrawMainMenu — заглавный экран с героем и списком пунктов.
func (r *Renderer) DrawMainMenu(screen *ebiten.Image, selected int, message string) {
	r.drawMenuHeader(screen, message)
	r.DrawLanguageControls(screen)
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

	r.secondaryCaption(screen, "EMPTY NAME = ANONYMOUS", float64(box.Max.Y+26))
	if len(status) > 0 {
		r.drawStatusLines(screen, status[0], float64(box.Max.Y+58), r.secondaryFace())
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

// DrawEndScreen — экран смерти или победы.
func (r *Renderer) DrawEndScreen(screen *ebiten.Image, title, submitStatus string) {
	l := r.Layout
	r.backdrop(screen)
	r.titleWithShadow(screen, title, float64(l.ScreenH)/2-80)

	if submitStatus != "" {
		r.drawStatusLines(screen, submitStatus, float64(l.ScreenH)/2+20, r.secondaryFace())
	}
}

func (r *Renderer) drawStatusLines(screen *ebiten.Image, status string, y float64, face text.Face) {
	for i, line := range r.statusLines(status, face) {
		r.TextCentered(screen, line, face, y+float64(i)*(TextWidth("M", face)+6), uiAccent)
	}
}

func (r *Renderer) statusLines(status string, face text.Face) []string {
	status = r.translateMessage(status)
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
		lines = append(lines, it.Name+locale.StatSuffix(l.Language, it.StatLabel()))
	}

	rowH := 22.0
	face := r.Fonts.UI
	if l.Touch {
		rowH, face = 34, r.Fonts.Compact
	}

	boxH := 10 + float64(len(lines))*rowH
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
}

func QuitDialogBounds(l Layout) image.Rectangle {
	w, h := min(440, l.ScreenW-32), 344
	x, y := (l.ScreenW-w)/2, (l.ScreenH-h)/2
	return image.Rect(x, y, x+w, y+h)
}

// DrawQuitDialog — подтверждение выхода в главное меню.
func (r *Renderer) DrawQuitDialog(screen *ebiten.Image, selected int) {
	r.dimScreen(screen)
	box := QuitDialogBounds(r.Layout)
	r.stonePanel(screen, box)
	y := float64(box.Min.Y)
	r.TextCentered(screen, r.tr("LEAVE RUN?"), r.Fonts.Menu, y+28, uiAccent)
	r.TextCentered(screen, r.tr("This run will be lost."), r.Fonts.UI, y+78, uiText)
	r.DrawMenuButtons(screen, MenuQuit, selected)
}

// dimScreen затемняет кадр под диалогом.
func (r *Renderer) dimScreen(screen *ebiten.Image) {
	l := r.Layout
	vector.DrawFilledRect(screen, 0, 0, float32(l.ScreenW), float32(l.ScreenH), overlayDim, false)
}
