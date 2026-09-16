package render

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
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

func TestDesktopHelpPanelFitsContent(t *testing.T) {
	l := DesktopLayout()
	_, height := helpContent(l)
	box := HelpViewBounds(l)
	if box.Dy() != height+32 {
		t.Fatal("desktop help panel should fit its content with standard padding")
	}
	back := MenuButtons(l, MenuHelp)[0].Bounds
	if back.Min.Y != box.Max.Y+20 || back.Max.Y > l.ScreenH-24 {
		t.Fatal("BACK should follow the compact help panel and stay on screen")
	}
	for _, mobile := range []Layout{TouchLayout(390, 600), TouchLayout(390, 844)} {
		if HelpViewBounds(mobile) != helpAvailableBounds(mobile) {
			t.Fatal("desktop compact layout must not change mobile help bounds")
		}
	}
}

func TestMobileHelpSectionSpacingUsesBottomSlack(t *testing.T) {
	l := TouchLayout(390, 844)
	rows, height := helpContent(l)
	for i, row := range rows {
		if row.section != "" && rows[i+1].y-row.y != 34 {
			t.Fatal("mobile section heading needs extra space before its first entry")
		}
	}
	last := rows[len(rows)-1]
	size := mobileHelpSize(l)
	wantHeight := size + 10 + len(last.lines)*(size+4)
	if last.height != wantHeight {
		t.Fatal("extra space should not be added below the last entry")
	}
	if height != HelpViewBounds(l).Inset(16).Dy() || HelpScrollLimit(l) != 0 {
		t.Fatal("spacing must preserve a single-screen help layout")
	}
}

func TestReferenceScreensFitWithoutDroppingContent(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844), TouchLayout(844, 390)} {
		rows, height := helpContent(l)
		view := HelpViewBounds(l).Inset(16)
		columns, face := 1, fonts.UI
		nameFace := fonts.Menu
		if l.Touch {
			face, nameFace = fonts.Compact, fonts.UI
			if mobileHelpSize(l) == 14 {
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
		expectedEntries = append(expectedEntries, helpEnemies...)
		expectedEntries = append(expectedEntries, helpItems...)
		for _, row := range rows {
			if row.section != "" {
				sections++
				continue
			}
			if row.entry != expectedEntries[entries] {
				t.Fatal("help must preserve full desktop descriptions")
			}
			entries++
			if TextWidth(strings.ToUpper(row.entry.name), nameFace) > float64(nameWidth) {
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
		if entries != len(helpEnemies)+len(helpItems) || sections != 2 {
			t.Fatal("help does not include both sections")
		}
		if HelpScrollLimit(l) != max(0, height-view.Dy()) {
			t.Fatal("help scroll range wrong")
		}
		if HelpScrollLimit(l) != 0 {
			t.Fatal("help should fit without scrolling on desktop and mobile")
		}
		if l.Touch && height != view.Dy() {
			t.Fatal("mobile help should use the available panel height")
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
