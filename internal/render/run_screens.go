package render

import (
	"fmt"
	"image"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
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
	items := "Pick up loot by walking over it. H/J/K/E: weapons, food, clarities, scrolls."
	if touch {
		move = "Use the D-pad. Step into an enemy to strike with your sword."
		items = "Pick up loot by walking over it. Tap an item category, then SELECT to use it."
	}
	return []helpEntry{
		{"portal", "REACH THE PORTAL", fmt.Sprintf("Find the portal on each level. Escape level %d to win. Gold is your score.", domain.MaxLevels)},
		{"player", "MOVE & ATTACK", move},
		{"food", "COLLECT & USE", items},
		{"elixir", "TAKE YOUR TIME", "Enemies act when you take a turn. Food heals, clarity buffs are temporary, scrolls are permanent. MENU pauses; HELP has details."},
	}
}

type welcomeRow struct {
	entry     helpEntry
	lines     []string
	y, height int
}

type welcomeTextLayout struct {
	face       text.Face
	lineHeight int
	rows       []welcomeRow
}

// Fit the largest shared font, then distribute space between actual text blocks.
func (r *Renderer) welcomeText(box image.Rectangle) welcomeTextLayout {
	entries := welcomeEntries(r.Layout.Touch)
	available := box.Dy() - 32
	var layout welcomeTextLayout
	for _, face := range []text.Face{r.Fonts.Menu, r.Fonts.UI, r.Fonts.Compact, r.Fonts.Small} {
		layout = welcomeTextLayout{face: face, lineHeight: int(TextWidth("M", face)) + 4}
		chars := max(1, int(float64(box.Dx()-88)/TextWidth("M", face)))
		total := 0
		for _, entry := range entries {
			lines := wrapText(entry.desc, chars)
			height := 30 + len(lines)*layout.lineHeight
			layout.rows = append(layout.rows, welcomeRow{entry: entry, lines: lines, height: height})
			total += height
		}
		if total+12*(len(entries)-1) > available && face != r.Fonts.Small {
			continue
		}
		gap := max(0, (available-total)/(len(entries)-1))
		y := box.Min.Y + 16
		for i := range layout.rows {
			layout.rows[i].y = y
			y += layout.rows[i].height + gap
		}
		break
	}
	return layout
}

func (r *Renderer) DrawWelcome(screen *ebiten.Image) {
	r.backdrop(screen)
	r.titleWithShadow(screen, "WELCOME", 36)
	r.secondaryCaption(screen, "EVERY STEP COUNTS", 98)
	box := runPanelBounds(r.Layout, false)
	r.stonePanel(screen, box)
	layout := r.welcomeText(box)
	x := float64(box.Min.X + 68)
	for _, row := range layout.rows {
		y := float64(row.y)
		r.drawFitted(screen, row.entry.role, float64(box.Min.X+18), y+6, 36)
		r.Text(screen, row.entry.name, r.Fonts.Menu, x, y, uiAccent)
		for j, line := range row.lines {
			r.Text(screen, line, layout.face, x, y+30+float64(j*layout.lineHeight), uiText)
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
		r.Text(screen, stat.label, r.Fonts.Compact, x, y, uiSecondary)
		r.Text(screen, fitLabel(stat.value, face, float64(colW)), face, x, y+24, uiText)
	}
	fillBox(screen, image.Rect(box.Min.X+20, box.Min.Y+90, box.Max.X-20, box.Min.Y+91), uiAccent)
	face := r.secondaryFace()
	chars := int(float64(box.Dx()) / TextWidth("M", face))
	for i, line := range wrapText(status, chars) {
		r.TextCentered(screen, line, face, float64(box.Max.Y+16)+float64(i)*(TextWidth("M", face)+6), uiAccent)
	}
}
