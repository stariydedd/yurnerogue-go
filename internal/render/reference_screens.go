package render

import (
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func helpAvailableBounds(l Layout) image.Rectangle {
	w := min(1160, l.ScreenW-32)
	return image.Rect((l.ScreenW-w)/2, 104, (l.ScreenW+w)/2, l.ScreenH-108)
}

// HelpPages are the help screens in paging order.
var HelpPages = []MenuPage{MenuHelp, MenuGlossary}

// HelpViewBounds is shared by both help pages, so switching pages never moves
// the panel or its buttons.
func HelpViewBounds(l Layout) image.Rectangle {
	box := helpAvailableBounds(l)
	if !l.Touch {
		height := 0
		for _, page := range HelpPages {
			_, pageHeight := helpContent(l, page)
			height = max(height, pageHeight)
		}
		box.Max.Y = min(box.Max.Y, box.Min.Y+height+32)
	}
	return box
}

type helpContentRow struct {
	entry        helpEntry
	section      string
	x, y, height int
	iconW        int
	lines        []string
}

func mobileHelpRowGap(size int) int {
	if size == 12 {
		return 0
	}
	return 2
}

// helpGroup is one help section; on wide screens column picks its column.
type helpGroup struct {
	section string
	entries []helpEntry
	column  int
}

func helpGroups(l Layout, page MenuPage) []helpGroup {
	if page == MenuGlossary {
		return []helpGroup{{"ENEMIES", helpEnemies, 0}, {"ITEMS", helpItems, 1}}
	}
	if l.Touch {
		return []helpGroup{{"BUTTONS", helpTouchControls, 0}, {"ABILITIES", helpTouchAbilities, 1}}
	}
	return []helpGroup{{"KEYS", helpDesktopControls, 0}, {"ABILITIES", helpDesktopAbilities, 1}, {"INTERFACE", helpDesktopInterface, 1}}
}

func mobileHelpSize(l Layout, page MenuPage) int {
	view := helpAvailableBounds(l).Inset(16)
	groups := helpGroups(l, page)
	for _, size := range []int{14, 12} {
		height := 22 * len(groups)
		for _, group := range groups {
			for _, entry := range group.entries {
				height += size + 8 + mobileHelpRowGap(size) + len(wrapText(locale.Text(l.Language, entry.desc), view.Dx()/size))*(size+4)
			}
		}
		if height <= view.Dy() {
			return size
		}
	}
	return 12
}

func helpContent(l Layout, page MenuPage) ([]helpContentRow, int) {
	view := helpAvailableBounds(l).Inset(16)
	groups := helpGroups(l, page)
	columns, size := 1, 14
	headingH, rowGap, sectionH, groupGap := 28, 16, 38, 16
	if l.Touch {
		size = mobileHelpSize(l, page)
		headingH, rowGap, sectionH, groupGap = size+8, mobileHelpRowGap(size), 22, 0
	}
	if l.ScreenW >= 900 {
		columns, size = 2, 18
	}
	width := (view.Dx() - (columns-1)*24) / columns
	chars := max(1, (width-52)/size)
	if l.Touch {
		chars = max(1, width/size)
	}
	var rows []helpContentRow
	y, total, count := 0, 0, 0
	for i, group := range groups {
		x := 0
		if columns == 2 {
			x = group.column * (width + 24)
		}
		switch {
		case columns == 2 && (i == 0 || group.column != groups[i-1].column):
			y = 0
		case i > 0:
			y += groupGap
		}
		rows = append(rows, helpContentRow{section: group.section, x: x, y: y, height: sectionH})
		y += sectionH
		count += len(group.entries)
		iconW, groupChars := 52, chars
		if !l.Touch {
			iconW = helpIconWidth(groups, group.column, columns)
			groupChars = max(1, (width-iconW)/size)
		}
		for _, entry := range group.entries {
			lines := wrapText(locale.Text(l.Language, entry.desc), groupChars)
			height := max(headingH+len(lines)*(size+4), helpGraphicHeight(entry)) + rowGap
			rows = append(rows, helpContentRow{entry: entry, x: x, y: y, height: height, lines: lines, iconW: iconW})
			y += height
		}
		total = max(total, y)
	}
	if l.Touch && total < view.Dy() {
		extra, added, entryIndex := view.Dy()-total, 0, 0
		sectionBonus := min(12, extra/len(groups))
		extra -= len(groups) * sectionBonus
		for i := range rows {
			rows[i].y += added
			if rows[i].section != "" {
				rows[i].height += sectionBonus
				added += sectionBonus
			} else {
				bonus := 0
				if entryIndex < count-1 {
					bonus = extra / (count - 1)
				}
				if entryIndex < extra%(count-1) {
					bonus++
				}
				rows[i].height += bonus
				added += bonus
				entryIndex++
			}
		}
		total += added
	}
	return rows, total
}

// Help keycaps match the HUD key badges.
const (
	helpKeyW, helpKeyH, helpKeyGap = 34, 30, 3
	helpSlotSize                   = 56
)

func helpKeyWidth(key string) int { return max(helpKeyW, 16*len(key)+14) }

// helpGraphicWidth is how wide the icon or keys of an entry are drawn.
func helpGraphicWidth(entry helpEntry) int {
	switch {
	case entry.keys == "":
		return 36
	case entry.role != "":
		return helpSlotSize
	}
	width := 0
	for _, row := range helpKeyRows(entry.keys) {
		rowW := -helpKeyGap
		for _, key := range row {
			rowW += helpKeyWidth(key) + helpKeyGap
		}
		width = max(width, rowW)
	}
	return width
}

// helpIconWidth is the icon column of one help column: wide enough for its
// widest icon or key cluster, so a WASD cluster widens only its own column.
func helpIconWidth(groups []helpGroup, column, columns int) int {
	width := 36
	for _, group := range groups {
		if columns == 1 || group.column == column {
			for _, entry := range group.entries {
				width = max(width, helpGraphicWidth(entry))
			}
		}
	}
	return width + 16
}

func helpKeyRows(keys string) [][]string {
	var rows [][]string
	for _, row := range strings.Split(keys, "/") {
		rows = append(rows, strings.Fields(row))
	}
	return rows
}

// helpGraphicHeight is how tall the keys column of an entry is drawn, so a
// short description never lets the next row overlap it.
func helpGraphicHeight(entry helpEntry) int {
	switch {
	case entry.keys == "":
		return 0
	case entry.role != "":
		return 4 + helpSlotSize + helpKeyH/2
	default:
		rows := len(helpKeyRows(entry.keys))
		return 4 + rows*helpKeyH + (rows-1)*helpKeyGap
	}
}

// drawHelpKeys draws the keys of an entry centred on cx: an icon in a HUD slot
// with its key badge underneath, or a cluster of keycaps.
func (r *Renderer) drawHelpKeys(dst *ebiten.Image, entry helpEntry, cx, y int) {
	top := y + 4
	if entry.role != "" {
		slot := image.Rect(cx-helpSlotSize/2, top, cx+helpSlotSize/2, top+helpSlotSize)
		r.uiSlot(dst, slot, false)
		r.drawFitted(dst, entry.role, float64(slot.Min.X+8), float64(slot.Min.Y+6), helpSlotSize-16)
		top = slot.Max.Y - helpKeyH/2
	}
	for i, row := range helpKeyRows(entry.keys) {
		widths, total := make([]int, len(row)), -helpKeyGap
		for j, key := range row {
			widths[j] = helpKeyWidth(key)
			total += widths[j] + helpKeyGap
		}
		kx, ky := cx-total/2, top+i*(helpKeyH+helpKeyGap)
		for j, key := range row {
			keycap := image.Rect(kx, ky, kx+widths[j], ky+helpKeyH)
			r.uiSlot(dst, keycap, false)
			lx, ly := float64(keycap.Min.X+keycap.Dx()/2)-TextWidth(key, r.Fonts.Menu)/2, float64(keycap.Min.Y+keycap.Dy()/2-9)
			r.Text(dst, key, r.Fonts.Menu, lx+1, ly+1, uiInk)
			r.Text(dst, key, r.Fonts.Menu, lx, ly, uiText)
			kx += widths[j] + helpKeyGap
		}
	}
}

func HelpScrollLimit(l Layout, page MenuPage) int {
	_, height := helpContent(l, page)
	return max(0, height-HelpViewBounds(l).Inset(16).Dy())
}

// HelpTitle names a help page; it is also the label of the button leading to it.
func HelpTitle(page MenuPage) string {
	if page == MenuGlossary {
		return "GLOSSARY"
	}
	return "CONTROLS"
}

func (r *Renderer) DrawHelp(screen *ebiten.Image, page MenuPage, scrollArg ...int) {
	r.backdrop(screen)
	r.titleWithShadow(screen, HelpTitle(page), 32)
	box := HelpViewBounds(r.Layout)
	r.stonePanel(screen, box)
	view := box.Inset(16)
	clip := screen.SubImage(view).(*ebiten.Image)
	scroll := 0
	if len(scrollArg) > 0 {
		scroll = max(0, min(HelpScrollLimit(r.Layout, page), scrollArg[0]))
	}
	rows, height := helpContent(r.Layout, page)
	face := r.Fonts.UI
	nameFace, headingH := r.Fonts.Menu, 28
	if r.Layout.Touch {
		size := mobileHelpSize(r.Layout, page)
		face, nameFace, headingH = r.Fonts.Compact, r.Fonts.UI, size+8
		if size == 14 {
			face, nameFace = r.Fonts.UI, r.Fonts.Menu
		}
	}
	if r.Layout.ScreenW >= 900 {
		face = r.Fonts.Menu
	}
	lineH := int(TextWidth("M", face)) + 4
	for _, row := range rows {
		y := view.Min.Y + row.y - scroll
		if y+row.height < view.Min.Y || y >= view.Max.Y {
			continue
		}
		x := view.Min.X + row.x
		if row.section != "" {
			r.Text(clip, r.tr(row.section), r.Fonts.Menu, float64(x), float64(y), uiAccent)
			continue
		}
		iconW := max(52, row.iconW)
		nameX, descX := x+iconW, x+iconW
		switch {
		case row.entry.keys != "" && !r.Layout.Touch:
			r.drawHelpKeys(clip, row.entry, x+(iconW-16)/2, y)
		case row.entry.role == "":
			// Key bindings have no icon: the key itself is the heading.
			nameX = x
			if r.Layout.Touch {
				descX = x
			}
		case r.Layout.Touch:
			r.drawFitted(clip, row.entry.role, float64(x), float64(y-4), float64(headingH+4))
			nameX, descX = x+34, x
		default:
			r.drawFitted(clip, row.entry.role, float64(x), float64(y+4), 36)
		}
		r.Text(clip, strings.ToUpper(helpEntryName(r.Layout, row.entry)), nameFace, float64(nameX), float64(y), uiAccent)
		for j, line := range row.lines {
			r.Text(clip, line, face, float64(descX), float64(y+headingH+j*lineH), uiText)
		}
	}
	drawReferenceScrollbar(screen, image.Rect(box.Max.X-10, view.Min.Y, box.Max.X-6, view.Max.Y), height, scroll)
}

func LeaderboardViewBounds(l Layout) image.Rectangle {
	w := min(960, l.ScreenW-32)
	bottom := l.ScreenH - 108
	if leaderboardDetailed(l) {
		w = min(1080, l.ScreenW-32)
		bottom = min(bottom, 136+424)
	}
	return image.Rect((l.ScreenW-w)/2, 136, (l.ScreenW+w)/2, bottom)
}

const leaderboardVisibleRows = 10

func leaderboardDetailed(l Layout) bool { return l.ScreenW >= 1100 }

type leaderboardColumn struct {
	label    string
	x, width int
}

func leaderboardColumns(l Layout) []leaderboardColumn {
	box := LeaderboardViewBounds(l).Inset(16)
	if leaderboardDetailed(l) {
		labels := []string{"#", "NAME", "GOLD", "LVL", "KILLS", "ATK", "HIT", "STEPS", "FOOD", "CLARITY", "SCROLL"}
		// Leave a full character of space between even the longest headings.
		const gap = 15
		chars := []int{2, 16, 6, 3, 5, 5, 5, 5, 4, 7, 6}
		used := (len(labels) - 1) * gap
		for _, count := range chars {
			used += count * 14
		}
		columns := make([]leaderboardColumn, 0, len(labels))
		x := box.Min.X
		for i, label := range labels {
			width := chars[i] * 14
			if i == 1 {
				width += box.Dx() - used
			}
			columns = append(columns, leaderboardColumn{label, x, width})
			x += width + gap
		}
		return columns
	}
	unit := 18
	if l.ScreenW < 900 {
		unit = 12
	}
	rankW, goldW, levelW := unit*2+8, unit*7, unit*3
	nameW := box.Dx() - rankW - goldW - levelW - 24
	x := box.Min.X
	columns := []leaderboardColumn{{"#", x, rankW}, {"NAME", x + rankW + 8, nameW}, {"GOLD", box.Max.X - levelW - 8 - goldW, goldW}, {"LVL", box.Max.X - levelW, levelW}}
	return columns
}

func leaderboardCellX(column leaderboardColumn, textWidth float64) float64 {
	x := float64(column.x)
	if column.label != "NAME" {
		x += (float64(column.width) - textWidth) / 2
	}
	return x
}

func leaderboardHeading(l Layout, column leaderboardColumn) string {
	if l.Language == locale.Russian && column.label == "KILLS" {
		return "УБИТО"
	}
	return locale.Text(l.Language, column.label)
}

func leaderboardValues(place int, rec LeaderboardRecord, detailed ...bool) []string {
	values := []string{fmt.Sprint(place), playerLabel(rec.PlayerName), fmt.Sprint(rec.Treasures), fmt.Sprint(rec.Level)}
	if len(detailed) > 0 && detailed[0] {
		for _, value := range []int{rec.EnemiesKilled, rec.AttacksMade, rec.HitsTaken, rec.TilesMoved, rec.FoodUsed, rec.ElixirsUsed, rec.ScrollsRead} {
			values = append(values, fmt.Sprint(value))
		}
	}
	return values
}

func leaderboardRowHeight(l Layout) int {
	return (LeaderboardViewBounds(l).Dy() - 64) / leaderboardVisibleRows
}

func (r *Renderer) DrawLeaderboard(screen *ebiten.Image, records []LeaderboardRecord, loading bool, source string) {
	r.backdrop(screen)
	r.titleWithShadow(screen, "LEADERBOARD", 32)
	r.secondaryCaption(screen, source, 96)
	box := LeaderboardViewBounds(r.Layout)
	r.stonePanel(screen, box)
	if loading || len(records) == 0 {
		label := "NO RECORDS YET"
		if loading {
			label = "LOADING..."
		} else if source == "SERVER UNAVAILABLE" {
			label = "SERVER UNAVAILABLE"
		}
		r.TextCentered(screen, r.tr(label), r.Fonts.Menu, float64(box.Min.Y+box.Dy()/2-10), uiText)
		return
	}
	face := r.Fonts.Menu
	if leaderboardDetailed(r.Layout) {
		face = r.Fonts.UI
	} else if r.Layout.ScreenW < 900 {
		face = r.Fonts.Compact
	}
	columns := leaderboardColumns(r.Layout)
	drawCell := func(value string, column leaderboardColumn, y float64, highlight bool) {
		value = fitLabel(value, face, float64(column.width))
		x := leaderboardCellX(column, TextWidth(value, face))
		clr := uiText
		if highlight {
			clr = uiAccent
		}
		r.Text(screen, value, face, x, y, clr)
	}
	for _, column := range columns {
		label := leaderboardHeading(r.Layout, column)
		drawCell(label, column, float64(box.Min.Y+18), true)
	}
	fillBox(screen, image.Rect(box.Min.X+16, box.Min.Y+46, box.Max.X-16, box.Min.Y+47), uiEdge)
	rowH := leaderboardRowHeight(r.Layout)
	for index, rec := range records[:min(len(records), leaderboardVisibleRows)] {
		y := box.Min.Y + 54 + index*rowH
		if index%2 == 0 {
			fillBox(screen, image.Rect(box.Min.X+12, y, box.Max.X-12, y+rowH), uiRecess)
		}
		values := leaderboardValues(index+1, rec, leaderboardDetailed(r.Layout))
		textY := float64(y) + (float64(rowH)-TextWidth("M", face))/2
		for i, column := range columns {
			drawCell(values[i], column, textY, index == 0)
		}
	}
}

func drawReferenceScrollbar(screen *ebiten.Image, track image.Rectangle, contentHeight, scroll int) {
	if contentHeight <= track.Dy() {
		return
	}
	fillBox(screen, track, uiRecess)
	height := max(24, track.Dy()*track.Dy()/contentHeight)
	top := track.Min.Y + (track.Dy()-height)*scroll/(contentHeight-track.Dy())
	fillBox(screen, image.Rect(track.Min.X, top, track.Max.X, top+height), uiAccent)
}
