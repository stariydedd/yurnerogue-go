package render

import (
	"fmt"
	"image"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func runPanelBounds(l Layout, results bool) image.Rectangle {
	w := min(680, l.ScreenW-32)
	bottom := l.ScreenH - 198
	if results {
		bottom = l.ScreenH - 242
	}
	return image.Rect((l.ScreenW-w)/2, 144, (l.ScreenW+w)/2, bottom)
}

func welcomeEntries(touch bool) []helpEntry {
	move := "Use WASD or arrows. Step into an enemy to strike with your sword."
	items := "Stronger weapons equip, weaker ones sharpen yours. Scrolls apply on pickup. C: food. X: potions."
	combat := "F, then direction: critical strike x1.5, recharges after 3 attacks. E next to an enemy: parry the hit and strike back."
	if touch {
		move = "Use the D-pad. Step into an enemy to strike with your sword."
		items = "Stronger weapons equip, weaker ones sharpen yours. Scrolls apply on pickup. Select food or potions to use."
		combat = "Critical Strike, then direction: damage x1.5, recharges after 3 attacks. Parry next to an enemy: blocks the hit and strikes back."
	}
	return []helpEntry{
		{"portal", "REACH THE PORTAL", fmt.Sprintf("Find the portal on each level. Escape level %d to win. Gold is your score.", domain.MaxLevels), ""},
		{"player", "MOVE & ATTACK", move, ""},
		{"food", "COLLECT & USE", items, ""},
		{"special-strike", "ABILITIES", combat, ""},
	}
}

type welcomeRow struct {
	entry     helpEntry
	lines     []string
	y, height int
}

type welcomeTextLayout struct {
	panel         image.Rectangle
	face          text.Face
	lineHeight    int
	headingHeight int
	gap           int
	rows          []welcomeRow
}

// Выбор шрифта прежний, но каждый блок подгоняется под его настоящий текст.
func (r *Renderer) welcomeText(box image.Rectangle) welcomeTextLayout {
	entries := welcomeEntries(r.Layout.Touch)
	available := box.Dy() - 32
	var layout welcomeTextLayout
	for _, face := range []text.Face{r.Fonts.Menu, r.Fonts.UI, r.Fonts.Compact, r.Fonts.Small} {
		headingHeight, gap := 24, 12
		if face == r.Fonts.Compact || face == r.Fonts.Small {
			gap = 8
		}
		layout = welcomeTextLayout{face: face, lineHeight: int(TextWidth("M", face)) + 2, headingHeight: headingHeight, gap: gap}
		chars := max(1, int(float64(box.Dx()-88)/TextWidth("M", face)))
		total := 0
		for _, entry := range entries {
			lines := wrapText(r.tr(entry.desc), chars)
			lineCount := max(len(wrapText(entry.desc, chars)), len(wrapText(locale.Text(locale.Russian, entry.desc), chars)))
			height := headingHeight + len(lines)*layout.lineHeight
			layout.rows = append(layout.rows, welcomeRow{entry: entry, lines: lines, height: height})
			total += headingHeight + lineCount*layout.lineHeight
		}
		if total+gap*(len(entries)-1) > available && face != r.Fonts.Small {
			continue
		}
		y := box.Min.Y + 16
		for i := range layout.rows {
			layout.rows[i].y = y
			y += layout.rows[i].height + gap
		}
		layout.panel = box
		layout.panel.Max.Y = y - gap + 16
		break
	}
	return layout
}

func (r *Renderer) DrawWelcome(screen *ebiten.Image) {
	r.backdrop(screen)
	r.titleWithShadow(screen, "WELCOME", 36)
	r.secondaryCaption(screen, "EVERY STEP COUNTS", 98)
	box := runPanelBounds(r.Layout, false)
	layout := r.welcomeText(box)
	r.stonePanel(screen, layout.panel)
	x := float64(box.Min.X + 68)
	for _, row := range layout.rows {
		y := float64(row.y)
		r.drawFitted(screen, row.entry.role, float64(box.Min.X+18), y+6, 36)
		r.Text(screen, r.tr(row.entry.name), r.Fonts.Menu, x, y, uiAccent)
		for j, line := range row.lines {
			r.Text(screen, line, layout.face, x, y+float64(layout.headingHeight+j*layout.lineHeight), uiText)
		}
	}
}

type runStat struct {
	label, value string
}

func runSummaryStats(s *domain.Session) []runStat {
	var stats domain.Stats
	level, gold, turns := 1, 0, 0
	if s != nil {
		stats, turns = s.Stats, s.Turns
		level = max(1, min(s.LevelNum, domain.MaxLevels))
		if s.Player != nil {
			gold = s.Player.Treasures
		}
	}
	return []runStat{
		{"GOLD", strconv.Itoa(gold)},
		{"LEVEL", fmt.Sprintf("%d / %d", level, domain.MaxLevels)},
		{"KILLS", strconv.Itoa(stats.EnemiesKilled)},
		{"TURNS", strconv.Itoa(turns)},
		{"ATTACKS", strconv.Itoa(stats.AttacksMade)},
		{"HITS TAKEN", strconv.Itoa(stats.HitsTaken)},
		{"STEPS", strconv.Itoa(stats.TilesMoved)},
		{"FOOD USED", strconv.Itoa(stats.FoodUsed)},
		{"CLARITIES", strconv.Itoa(stats.ElixirsUsed)},
		{"SCROLLS", strconv.Itoa(stats.ScrollsRead)},
	}
}

// Placement: место завершённого забега в таблице рекордов. Place 0 означает,
// что место неизвестно. GoldShort за пределами топ-10: золото, которого
// не хватило, чтобы туда попасть.
type Placement struct {
	Place, GoldShort int
}

var (
	deathRed    = color.RGBA{168, 24, 24, 255}
	placeGold   = color.RGBA{255, 204, 51, 255}
	placeSilver = color.RGBA{206, 214, 224, 255}
	placeBronze = color.RGBA{205, 127, 50, 255}
)

// placeColor даёт пьедесталу медали; остальная часть топ-10 обычная,
// а места ниже приглушены.
func placeColor(place int) color.Color {
	switch {
	case place == 1:
		return placeGold
	case place == 2:
		return placeSilver
	case place == 3:
		return placeBronze
	case place <= 10:
		return uiText
	}
	return uiMuted
}

func (r *Renderer) DrawRunSummary(screen *ebiten.Image, s *domain.Session, won bool, status string, place Placement) {
	r.backdrop(screen)
	box := runPanelBounds(r.Layout, true)
	hero := r.runHero(box)
	title, clr := "YOU DIED", color.Color(deathRed)
	if won {
		title, clr = "VICTORY", uiAccent
	}
	head := r.runHeader(title, place, hero)
	r.Text(screen, head.title, head.face, head.x+2, head.y+2, uiInk)
	r.Text(screen, head.title, head.face, head.x, head.y, clr)
	r.secondaryCaption(screen, "RUN SUMMARY", 98)
	r.stonePanel(screen, box)
	r.drawRunHero(screen, hero)
	r.drawPlace(screen, box, place, hero, head.numberSize)
	colW := (box.Dx() - 48) / 2
	rowH := (box.Dy() - 116) / 4
	for i, stat := range runSummaryStats(s) {
		x := float64(box.Min.X + 20 + (i%2)*(colW+8))
		y := float64(box.Min.Y + 20)
		face := r.Fonts.Menu
		if i >= 2 {
			y += float64(96 + ((i-2)/2)*rowH)
			face = r.Fonts.UI
		}
		label := fitLabel(r.tr(stat.label), r.Fonts.Compact, float64(colW))
		r.Text(screen, label, r.Fonts.Compact, x, y, uiSecondary)
		r.Text(screen, fitLabel(stat.value, face, float64(colW)), face, x, y+24, uiText)
	}
	fillBox(screen, image.Rect(box.Min.X+20, box.Min.Y+90, box.Max.X-20, box.Min.Y+91), uiAccent)
	face := r.secondaryFace()
	chars := int(float64(box.Dx()) / TextWidth("M", face))
	lines := wrapText(r.translateMessage(status), chars)
	if !hero.wide && place.GoldShort > 0 {
		// На телефоне рядом с панелью места нет: нехватка золота уходит под неё.
		lines = append(lines, wrapText(r.goldShort(place), chars)...)
	}
	lineH := int(TextWidth("M", face)) + 6
	top := statusTop(r.Layout, box, len(lines), lineH)
	for i, line := range lines {
		r.TextCentered(screen, line, face, float64(top+i*lineH), uiAccent)
	}
}

// statusTop ставит строки под панелью итогов на 16 px ниже неё или выше,
// по центру промежутка, если иначе они теснят кнопки.
func statusTop(l Layout, box image.Rectangle, lines, lineH int) int {
	gap := MenuButtons(l, MenuResults)[0].Bounds.Min.Y - box.Max.Y
	return box.Max.Y + max(4, min(16, (gap-lines*lineH)/2))
}

func (r *Renderer) goldShort(place Placement) string {
	return fmt.Sprintf(r.tr("%d gold short of the top 10"), place.GoldShort)
}

// phoneTitleGap держит заголовок на телефоне на таком расстоянии от героя и места;
// phoneMedal: размер шрифта места там, три четверти героя.
const phoneTitleGap, phoneMedal = 20, 48

// placeNumber: место в топ-10 в виде "#N"; на десктопе в рост героя,
// насколько позволяет столбец рядом с панелью, на телефоне phoneMedal;
// пусто для любого другого места.
func placeNumber(place Placement, hero runHero) (string, int) {
	if place.Place < 1 || place.Place > 10 {
		return "", 0
	}
	number := "#" + strconv.Itoa(place.Place)
	size := min(hero.h, phoneMedal)
	if hero.wide {
		size = min(hero.h, hero.placeRoom/len(number))
	}
	// Press Start 2P нарисован на сетке 8 px: целые кратные остаются чёткими.
	return number, max(8, size/8*8)
}

type runHeader struct {
	title      string
	face       text.Face
	x, y       float64
	numberSize int
}

// runHeader размещает заголовок по центру экрана. На телефоне он на уровне
// героя и не задевает ни его, ни место в топ-10.
func (r *Renderer) runHeader(title string, place Placement, hero runHero) runHeader {
	f := r.Fonts
	faces := []text.Face{f.Title, f.Sized(32), f.Sized(24), f.Menu}
	h := runHeader{title: r.tr(title), y: 36}
	pick := func(room int) bool {
		for _, face := range faces {
			if TextWidth(h.title, face) <= float64(room) {
				h.face = face
				return true
			}
		}
		h.face = faces[len(faces)-1]
		return false
	}
	number, size := placeNumber(place, hero)
	h.numberSize = size
	w := r.Layout.ScreenW
	if hero.wide {
		pick(w - 32)
		h.x = float64(w)/2 - TextWidth(h.title, h.face)/2
		return h
	}
	// Заголовок всегда оставляет место под однозначное место, поэтому
	// не меняет размер, когда место приходит с сервера через миг после открытия
	// экрана, и одинаков при любом месте; "#10" потом ужимается
	// в оставшееся рядом место.
	pick(w - 2*max(hero.x+hero.w+phoneTitleGap, 16+len("#1")*min(hero.h, phoneMedal)+phoneTitleGap))
	tw := TextWidth(h.title, h.face)
	if number != "" {
		room := (w-int(tw))/2 - 16 - phoneTitleGap
		h.numberSize = max(8, min(size, room/len(number))/8*8)
	}
	h.x = float64(w)/2 - tw/2
	h.y = float64(hero.y+hero.h/2) - TextWidth("M", h.face)/2
	return h
}

type runHero struct {
	img               *ebiten.Image
	scale, x, y, w, h int
	wide              bool // рядом с панелью; false: маленький, рядом с заголовком
	placeRoom         int  // ширина справа от панели, где стоит место
}

// runHero ставит Juggernaut рядом с итогами забега, в анимации ожидания,
// с целым масштабом, чтобы пиксель-арт оставался чётким. На десктопе он стоит слева
// от панели; на телефоне панель во всю ширину, поэтому там он меньше
// и стоит рядом с заголовком.
func (r *Renderer) runHero(panel image.Rectangle) runHero {
	img := r.sprites.Frame("player", r.AnimTick())
	size := image.Pt(32, 32)
	if img != nil {
		size = img.Bounds().Size()
	}
	room := panel.Min.X - 24 // свободная ширина слева от панели
	h := runHero{img: img, scale: min(6, room/max(1, size.X), panel.Dy()/max(1, size.Y))}
	h.x, h.y = (panel.Min.X-size.X*h.scale)/2, panel.Min.Y+(panel.Dy()-size.Y*h.scale)/2
	h.wide = h.scale >= 3
	if !h.wide {
		h.scale, h.x, h.y = 2, 16, 20
	}
	h.w, h.h = size.X*h.scale, size.Y*h.scale
	h.placeRoom = r.Layout.ScreenW - 24 - panel.Max.X
	return h
}

func (r *Renderer) drawRunHero(screen *ebiten.Image, h runHero) {
	if h.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(h.scale), float64(h.scale))
	op.GeoM.Translate(float64(h.x), float64(h.y))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(h.img, op)
}

// drawPlace зеркалит героя: справа от панели на десктопе, справа от
// заголовка на телефоне. Место в топ-10 это число в рост героя, на пьедестале
// в цветах медалей. Ниже показывается только золото, которого не хватило
// до топ-10, а неизвестное место не показывается.
func (r *Renderer) drawPlace(screen *ebiten.Image, panel image.Rectangle, place Placement, hero runHero, size int) {
	if place.Place < 1 || place.Place > 10 && (place.GoldShort < 1 || !hero.wide) {
		return // на телефоне нехватка золота уходит под панель
	}
	left := panel.Max.X + 12
	colW := r.Layout.ScreenW - 12 - left
	cx := float64(left) + float64(colW)/2
	cy := float64(hero.y + hero.h/2)
	centered := func(s string, face text.Face, y float64, clr color.Color) {
		x := cx - TextWidth(s, face)/2
		if !hero.wide {
			x = float64(r.Layout.ScreenW-16) - TextWidth(s, face) // вплотную к краю
		}
		r.Text(screen, s, face, x, y, clr)
	}
	if place.Place > 10 {
		face := r.Fonts.UI
		lines := wrapText(r.goldShort(place), max(1, int(float64(colW)/TextWidth("M", face))))
		lineH := TextWidth("M", face) + 10
		top := cy - lineH*float64(len(lines))/2
		for i, line := range lines {
			centered(line, face, top+float64(i)*lineH, uiSecondary)
		}
		return
	}
	number := "#" + strconv.Itoa(place.Place)
	top := cy - float64(size)/2
	if hero.wide {
		centered(fitLabel(r.tr("YOUR PLACE"), r.Fonts.Compact, float64(colW)), r.Fonts.Compact, top-32, uiSecondary)
	}
	centered(number, r.Fonts.Sized(size), top, placeColor(place.Place))
}
