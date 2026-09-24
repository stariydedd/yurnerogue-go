package render

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func TestLeaderboardOverviewHasScoreAndLevelWithoutVerificationMarker(t *testing.T) {
	rec := LeaderboardRecord{PlayerName: "tester", Treasures: 2484, Level: 14}
	want := []string{"1", "tester", "2484", "14"}
	if got := leaderboardValues(1, rec); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected table values: %v", got)
	}
}

func TestLeaderboardNamesAreSafe(t *testing.T) {
	for _, name := range []string{"stan", "Этофейкн", "fle", "лешадскиймальчик", "a\nb", "fle\xd0", "", strings.Repeat("я", 30)} {
		got := playerLabel(name)
		if !utf8.ValidString(got) || utf8.RuneCountInString(got) > 16 || strings.ContainsAny(got, "\n\r") {
			t.Fatalf("unsafe name: %q", got)
		}
	}
	if playerLabel("fle\xd0") != "fle" || playerLabel("a\nb") != "ab" || playerLabel("") != "anonymous" {
		t.Fatal("name sanitization changed")
	}
}

func TestDesktopLeaderboardIncludesAllMetrics(t *testing.T) {
	rec := LeaderboardRecord{PlayerName: "tester", Treasures: 2484, Level: 14, EnemiesKilled: 38, AttacksMade: 134, HitsTaken: 64, TilesMoved: 2143, FoodUsed: 18, ElixirsUsed: 9, ScrollsRead: 20}
	want := []string{"1", "tester", "2484", "14", "38", "134", "64", "2143", "18", "9", "20"}
	if got := leaderboardValues(1, rec, true); !reflect.DeepEqual(got, want) {
		t.Fatalf("incorrect desktop metrics: %v", got)
	}
	if len(leaderboardColumns(DesktopLayout())) != len(want) || len(leaderboardColumns(TouchLayout(390, 600))) != 4 {
		t.Fatal("desktop/mobile columns changed incorrectly")
	}
}

func TestDesktopLeaderboardIsCompact(t *testing.T) {
	l := DesktopLayout()
	box := LeaderboardViewBounds(l)
	if box.Dx() != 1080 || leaderboardRowHeight(l) != 36 {
		t.Fatal("desktop leaderboard is not compact")
	}
	back := MenuButtons(l, MenuLeaderboard)[0].Bounds
	if back.Min.Y != box.Max.Y+24 {
		t.Fatal("BACK detached from compact table")
	}
	mobile := TouchLayout(390, 600)
	if LeaderboardViewBounds(mobile).Max.Y != mobile.ScreenH-108 {
		t.Fatal("mobile layout changed")
	}
}

func TestLeaderboardLocalizedHeadersAndValuesAreAligned(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		face := fonts.UI
		if !leaderboardDetailed(l) {
			face = fonts.Compact
		}
		for _, language := range []locale.Language{locale.English, locale.Russian} {
			l.Language = language
			columns := leaderboardColumns(l)
			for i, column := range columns {
				heading := leaderboardHeading(l, column)
				if TextWidth(heading, face) > float64(column.width) {
					t.Fatalf("localized header clipped: %s", heading)
				}
				if i > 0 && leaderboardDetailed(l) && column.x-columns[i-1].x-columns[i-1].width < 14 {
					t.Fatal("desktop headings must have at least a character-wide gap")
				}
				for _, value := range []string{heading, "1", "10", "9999"} {
					width := TextWidth(fitLabel(value, face, float64(column.width)), face)
					x := leaderboardCellX(column, width)
					if column.label == "NAME" {
						if x != float64(column.x) {
							t.Fatal("names must stay left aligned")
						}
					} else if x+width/2 != float64(column.x)+float64(column.width)/2 {
						t.Fatal("numeric cells and their headings must share a center")
					}
				}
			}
		}
	}
}

func TestDesktopHelpPanelFitsContent(t *testing.T) {
	l := DesktopLayout()
	height := 0
	for _, page := range HelpPages {
		_, pageHeight := helpContent(l, page)
		height = max(height, pageHeight)
	}
	box := HelpViewBounds(l)
	if box.Dy() != height+32 {
		t.Fatal("desktop help panel should fit its taller page with standard padding")
	}
	for _, page := range HelpPages {
		buttons := MenuButtons(l, page)
		if len(buttons) != 2 || buttons[0].Action != "help-page" || buttons[0].Label != HelpTitle(OtherHelpPage(page)) || buttons[1].Action != "back" {
			t.Fatalf("%s must offer the other page and BACK", HelpTitle(page))
		}
		for _, button := range buttons {
			if button.Bounds.Min.Y != box.Max.Y+20 || button.Bounds.Max.Y > l.ScreenH-24 {
				t.Fatal("help buttons should follow the compact help panel and stay on screen")
			}
		}
		if buttons[0].Bounds.Overlaps(buttons[1].Bounds) {
			t.Fatal("help buttons overlap")
		}
	}
	for _, mobile := range []Layout{TouchLayout(390, 600), TouchLayout(390, 844)} {
		if HelpViewBounds(mobile) != helpAvailableBounds(mobile) {
			t.Fatal("desktop compact layout must not change mobile help bounds")
		}
	}
}

func TestMobileHelpSectionSpacingUsesBottomSlack(t *testing.T) {
	l := TouchLayout(390, 844)
	for _, page := range HelpPages {
		rows, height := helpContent(l, page)
		for i, row := range rows {
			if row.section != "" && rows[i+1].y-row.y != 34 {
				t.Fatal("mobile section heading needs extra space before its first entry")
			}
		}
		last := rows[len(rows)-1]
		size := mobileHelpSize(l, page)
		wantHeight := size + 10 + len(last.lines)*(size+4)
		if last.height != wantHeight {
			t.Fatal("extra space should not be added below the last entry")
		}
		if height != HelpViewBounds(l).Inset(16).Dy() || HelpScrollLimit(l, page) != 0 {
			t.Fatal("spacing must preserve a single-screen help layout")
		}
	}
}

func TestReferenceScreensFitWithoutDroppingContent(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844), TouchLayout(844, 390)} {
		for _, page := range HelpPages {
			rows, height := helpContent(l, page)
			view := HelpViewBounds(l).Inset(16)
			columns, face := 1, fonts.UI
			nameFace := fonts.Menu
			if l.Touch {
				face, nameFace = fonts.Compact, fonts.UI
				if mobileHelpSize(l, page) == 14 {
					face, nameFace = fonts.UI, fonts.Menu
				}
			}
			if l.ScreenW >= 900 {
				columns, face = 2, fonts.Menu
			}
			width := (view.Dx() - (columns-1)*24) / columns
			descWidth, nameWidth := width-52, width-52
			if l.Touch {
				descWidth, nameWidth = width, width-34
			}
			entries, sections := 0, 0
			var expectedEntries []helpEntry
			for _, group := range helpGroups(l, page) {
				expectedEntries = append(expectedEntries, group.entries...)
			}
			for _, row := range rows {
				if row.section != "" {
					sections++
					continue
				}
				if row.entry != expectedEntries[entries] {
					t.Fatal("help must preserve full desktop descriptions")
				}
				entries++
				rowNameWidth := nameWidth
				if row.entry.role == "" {
					rowNameWidth = width // key bindings have no icon column
				}
				if TextWidth(strings.ToUpper(row.entry.name), nameFace) > float64(rowNameWidth) {
					t.Fatal("help name clipped")
				}
				if row.y+row.height > height {
					t.Fatal("help row outside scrollable content")
				}
				if strings.Join(strings.Fields(strings.Join(row.lines, " ")), " ") != strings.Join(strings.Fields(row.entry.desc), " ") {
					t.Fatal("help description lost words")
				}
				for _, line := range row.lines {
					if TextWidth(line, face) > float64(descWidth) {
						t.Fatal("help line too wide")
					}
				}
			}
			if entries != len(expectedEntries) || sections != len(helpGroups(l, page)) {
				t.Fatal("help does not include every section")
			}
			if HelpScrollLimit(l, page) != max(0, height-view.Dy()) {
				t.Fatal("help scroll range wrong")
			}
			if HelpScrollLimit(l, page) != 0 {
				t.Fatalf("%s should fit without scrolling at %+v", HelpTitle(page), l)
			}
			if l.Touch && height != view.Dy() {
				t.Fatal("mobile help should use the available panel height")
			}
		}
		box := LeaderboardViewBounds(l)
		if box.Min.Y+54+leaderboardVisibleRows*leaderboardRowHeight(l) > box.Max.Y-10 {
			t.Fatal("top ten does not fit")
		}
		tableFace := fonts.Menu
		if leaderboardDetailed(l) {
			tableFace = fonts.UI
		} else if l.ScreenW < 900 {
			tableFace = fonts.Compact
		}
		if float64(leaderboardRowHeight(l)) < TextWidth("M", tableFace)+8 {
			t.Fatal("table rows overlap")
		}
		tableColumns := leaderboardColumns(l)
		for i, col := range tableColumns {
			if TextWidth(col.label, tableFace) > float64(col.width) {
				t.Fatalf("header clipped: %s", col.label)
			}
			if col.x < box.Min.X+16 || col.x+col.width > box.Max.X-16 {
				t.Fatal("column outside panel")
			}
			if i > 0 && col.x < tableColumns[i-1].x+tableColumns[i-1].width+8 {
				t.Fatal("table columns overlap")
			}
		}
		if TextWidth(strings.Repeat("я", 16), tableFace) > float64(tableColumns[1].width) {
			t.Fatal("full player name cannot fit")
		}
	}
}
