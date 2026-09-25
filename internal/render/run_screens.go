package render

import (
	"fmt"
	"image"
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

func (r *Renderer) DrawRunSummary(screen *ebiten.Image, s *domain.Session, name string, won bool, status string) {
	r.backdrop(screen)
	title := "YOU DIED"
	if won {
		title = "VICTORY"
	}
	r.titleWithShadow(screen, title, 36)
	r.secondaryCaption(screen, "RUN SUMMARY / "+playerLabel(name), 98)
	box := runPanelBounds(r.Layout, true)
	r.stonePanel(screen, box)
	if won {
		r.drawVictoryHero(screen, box)
	}
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
	for i, line := range wrapText(r.translateMessage(status), chars) {
		r.TextCentered(screen, line, face, float64(box.Max.Y+16)+float64(i)*(TextWidth("M", face)+6), uiAccent)
	}
}

// drawVictoryHero stands Juggernaut beside the results of a won run, idling,
// at a whole-number scale so the pixel art stays sharp. On desktop he stands
// left of the panel; the phone panel is full width, so there he stands
// smaller beside the title.
func (r *Renderer) drawVictoryHero(screen *ebiten.Image, panel image.Rectangle) {
	img := r.sprites.Frame("player", r.AnimTick())
	if img == nil {
		return
	}
	size := img.Bounds().Size()
	room := panel.Min.X - 24 // free width left of the panel
	scale := min(6, room/max(1, size.X), panel.Dy()/max(1, size.Y))
	x, y := (panel.Min.X-size.X*scale)/2, panel.Min.Y+(panel.Dy()-size.Y*scale)/2
	if scale < 3 {
		scale = 2
		x, y = 16, 20
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	op.GeoM.Translate(float64(x), float64(y))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(img, op)
}
