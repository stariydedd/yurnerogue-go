package render

import (
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

func helpAvailableBounds(l Layout) image.Rectangle {
	w := min(1160, l.ScreenW-32)
	return image.Rect((l.ScreenW-w)/2, 104, (l.ScreenW+w)/2, l.ScreenH-108)
}

func HelpViewBounds(l Layout) image.Rectangle {
	box := helpAvailableBounds(l)
	if !l.Touch {
		_, height := helpContent(l)
		box.Max.Y = min(box.Max.Y, box.Min.Y+height+32)
	}
	return box
}

type helpContentRow struct {
	entry        helpEntry
	section      string
	x, y, height int
	lines        []string
}

func mobileHelpSize(l Layout) int {
	view := HelpViewBounds(l).Inset(16)
	for _, size := range []int{14, 12} {
		height := 44
		for _, entries := range [][]helpEntry{helpEnemies, helpItems} {
			for _, entry := range entries {
				height += size + 10 + len(wrapText(entry.desc, view.Dx()/size))*(size+4)
			}
		}
		if height <= view.Dy() {
			return size
		}
	}
	return 12
}

func helpContent(l Layout) ([]helpContentRow, int) {
	view := helpAvailableBounds(l).Inset(16)
	columns, size := 1, 14
	headingH, rowGap, sectionH, groupGap := 28, 16, 38, 16
	if l.Touch {
		size = mobileHelpSize(l)
		headingH, rowGap, sectionH, groupGap = size+8, 2, 22, 0
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
	y, total := 0, 0
	for group, entries := range [][]helpEntry{helpEnemies, helpItems} {
		x := 0
		if columns == 2 {
			x = group * (width + 24)
			y = 0
		} else if group > 0 {
			y += groupGap
		}
		section := "ENEMIES"
		if group == 1 {
			section = "ITEMS"
		}
		rows = append(rows, helpContentRow{section: section, x: x, y: y, height: sectionH})
		y += sectionH
		for _, entry := range entries {
			lines := wrapText(entry.desc, chars)
			height := headingH + len(lines)*(size+4) + rowGap
			rows = append(rows, helpContentRow{entry: entry, x: x, y: y, height: height, lines: lines})
			y += height
		}
		total = max(total, y)
	}
	if l.Touch && total < view.Dy() {
		extra, added, entryIndex := view.Dy()-total, 0, 0
		count := len(helpEnemies) + len(helpItems)
		sectionBonus := min(12, extra/2)
		extra -= 2 * sectionBonus
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

func HelpScrollLimit(l Layout) int {
	_, height := helpContent(l)
	return max(0, height-HelpViewBounds(l).Inset(16).Dy())
}

func (r *Renderer) DrawHelp(screen *ebiten.Image, scrollArg ...int) {
	r.backdrop(screen)
	r.titleWithShadow(screen, "HELP", 32)
	box := HelpViewBounds(r.Layout)
	r.stonePanel(screen, box)
	view := box.Inset(16)
	clip := screen.SubImage(view).(*ebiten.Image)
	scroll := 0
	if len(scrollArg) > 0 {
		scroll = max(0, min(HelpScrollLimit(r.Layout), scrollArg[0]))
	}
	rows, height := helpContent(r.Layout)
	face := r.Fonts.UI
	nameFace, headingH := r.Fonts.Menu, 28
	if r.Layout.Touch {
		face, nameFace, headingH = r.Fonts.Compact, r.Fonts.UI, mobileHelpSize(r.Layout)+8
		if mobileHelpSize(r.Layout) == 14 {
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
			r.Text(clip, row.section, r.Fonts.Menu, float64(x), float64(y), uiAccent)
			continue
		}
		nameX, descX := x+52, x+52
		if r.Layout.Touch {
			r.drawFitted(clip, row.entry.role, float64(x), float64(y-4), float64(headingH+4))
			nameX, descX = x+34, x
		} else {
			r.drawFitted(clip, row.entry.role, float64(x), float64(y+4), 36)
		}
		r.Text(clip, strings.ToUpper(row.entry.name), nameFace, float64(nameX), float64(y), uiAccent)
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
		chars := []int{2, 16, 7, 3, 5, 6, 6, 6, 4, 7, 6}
		used := (len(labels) - 1) * 8
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
			x += width + 8
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
		r.TextCentered(screen, label, r.Fonts.Menu, float64(box.Min.Y+box.Dy()/2-10), uiText)
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
		x := float64(column.x)
		if column.label != "NAME" {
			x += float64(column.width) - TextWidth(value, face)
		}
		clr := uiText
		if highlight {
			clr = uiAccent
		}
		r.Text(screen, value, face, x, y, clr)
	}
	for _, column := range columns {
		drawCell(column.label, column, float64(box.Min.Y+18), true)
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
