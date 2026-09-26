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

// Keep the existing font selection, but size each block to its actual text.
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

// Placement is where a finished run landed on the leaderboard. Place 0 means
// the place is not known. GoldShort, outside the top 10, is the gold that
// would have got the run in.
type Placement struct {
	Place, GoldShort int
}

var (
	deathRed    = color.RGBA{168, 24, 24, 255}
	placeGold   = color.RGBA{255, 204, 51, 255}
	placeSilver = color.RGBA{206, 214, 224, 255}
	placeBronze = color.RGBA{205, 127, 50, 255}
)

// placeColor gives the podium its medals; the rest of the top 10 is plain
// and a place further down is muted.
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
		// The phone has no room beside the panel: the gap goes under it.
		lines = append(lines, wrapText(r.goldShort(place), chars)...)
	}
	lineH := int(TextWidth("M", face)) + 6
	top := statusTop(r.Layout, box, len(lines), lineH)
	for i, line := range lines {
		r.TextCentered(screen, line, face, float64(top+i*lineH), uiAccent)
	}
}

// statusTop puts the lines under the results panel 16 px below it, or higher,
// centered in the gap, when that would crowd the buttons.
func statusTop(l Layout, box image.Rectangle, lines, lineH int) int {
	gap := MenuButtons(l, MenuResults)[0].Bounds.Min.Y - box.Max.Y
	return box.Max.Y + max(4, min(16, (gap-lines*lineH)/2))
}

func (r *Renderer) goldShort(place Placement) string {
	return fmt.Sprintf(r.tr("%d gold short of the top 10"), place.GoldShort)
}

// phoneTitleGap keeps the phone title this far from the hero and the place;
// phoneMedal is the place's font size there, three quarters of the hero.
const phoneTitleGap, phoneMedal = 20, 48

// placeNumber is the top 10 place as drawn, "#N": on desktop as tall as the
// hero where the column beside the panel allows, on the phone phoneMedal;
// empty for any other place.
func placeNumber(place Placement, hero runHero) (string, int) {
	if place.Place < 1 || place.Place > 10 {
		return "", 0
	}
	number := "#" + strconv.Itoa(place.Place)
	size := min(hero.h, phoneMedal)
	if hero.wide {
		size = min(hero.h, hero.placeRoom/len(number))
	}
	// Press Start 2P is drawn on an 8 px grid: whole multiples stay crisp.
	return number, max(8, size/8*8)
}

type runHeader struct {
	title      string
	face       text.Face
	x, y       float64
	numberSize int
}

// runHeader lays out the title, centered on the screen. On the phone it is
// level with the hero and keeps clear of both him and a top 10 place.
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
	// The title always leaves room for a one digit place, so it keeps its
	// size when the place arrives from the server a moment after the screen
	// opens, and it is the same whatever the place; "#10" then shrinks into
	// the room left beside it.
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
	wide              bool // beside the panel; false: small, beside the title
	placeRoom         int  // width right of the panel, where the place goes
}

// runHero places Juggernaut beside the results of a run, idling, at a
// whole-number scale so the pixel art stays sharp. On desktop he stands left
// of the panel; the phone panel is full width, so there he stands smaller
// beside the title.
func (r *Renderer) runHero(panel image.Rectangle) runHero {
	img := r.sprites.Frame("player", r.AnimTick())
	size := image.Pt(32, 32)
	if img != nil {
		size = img.Bounds().Size()
	}
	room := panel.Min.X - 24 // free width left of the panel
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

// drawPlace mirrors the hero: right of the panel on desktop, right of the
// title on the phone. A top 10 place is a number as tall as the hero, in
// medal colors on the podium. Further down only the gold still missing for
// the top 10 is shown, and an unknown place shows nothing.
func (r *Renderer) drawPlace(screen *ebiten.Image, panel image.Rectangle, place Placement, hero runHero, size int) {
	if place.Place < 1 || place.Place > 10 && (place.GoldShort < 1 || !hero.wide) {
		return // on the phone the missing gold goes under the panel
	}
	left := panel.Max.X + 12
	colW := r.Layout.ScreenW - 12 - left
	cx := float64(left) + float64(colW)/2
	cy := float64(hero.y + hero.h/2)
	centered := func(s string, face text.Face, y float64, clr color.Color) {
		x := cx - TextWidth(s, face)/2
		if !hero.wide {
			x = float64(r.Layout.ScreenW-16) - TextWidth(s, face) // flush with the edge
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
