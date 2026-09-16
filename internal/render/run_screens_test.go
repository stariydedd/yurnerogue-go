package render

import (
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestRunScreensContentFits(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(320, 568), TouchLayout(390, 600), TouchLayout(430, 932), TouchLayout(844, 390)} {
		box := runPanelBounds(l, false)
		r := &Renderer{Layout: l, Fonts: fonts}
		layout := r.welcomeText(box)
		if TextWidth("M", layout.face) <= TextWidth("M", fonts.Small) {
			t.Fatal("welcome text was not enlarged")
		}
		for i, row := range layout.rows {
			if TextWidth(row.entry.name, fonts.Menu) > float64(box.Dx()-88) {
				t.Fatalf("welcome heading too wide: %s", row.entry.name)
			}
			if row.y < box.Min.Y+16 || row.y+row.height > box.Max.Y-16 {
				t.Fatalf("welcome description outside panel: %s", row.entry.name)
			}
			if i > 0 && row.y < layout.rows[i-1].y+layout.rows[i-1].height+12 {
				t.Fatal("welcome text blocks overlap")
			}
			for _, line := range row.lines {
				if TextWidth(line, layout.face) > float64(box.Dx()-88) {
					t.Fatalf("welcome line too wide: %s", line)
				}
			}
		}
		box = runPanelBounds(l, true)
		if (box.Dy()-116)/4 < 40 {
			t.Fatal("stats rows overlap")
		}
		for _, status := range []string{"Submitting score...", "Score submitted to global leaderboard!", "Score not submitted: leaderboard unavailable for this game.", "Game length limit reached: score cannot be submitted.", "Score submission could not be confirmed."} {
			face := r.secondaryFace()
			chars := int(float64(box.Dx()) / TextWidth("M", face))
			if box.Max.Y+16+len(wrapText(status, chars))*(int(TextWidth("M", face))+6) > MenuButtons(l, MenuResults)[0].Bounds.Min.Y {
				t.Fatal("submission status overlaps buttons")
			}
		}
		for _, page := range []MenuPage{MenuWelcome, MenuResults} {
			for i, button := range MenuButtons(l, page) {
				if menuButtonActive(page, button, i, 1) != (i == 1) {
					t.Fatal("wrong run menu selection highlighted")
				}
			}
		}
	}
}

func TestRunSummaryUsesRecordedStatsAndCapsWinningLevel(t *testing.T) {
	s := domain.NewSessionSeed(21)
	s.LevelNum, s.Turns, s.Player.Treasures = domain.MaxLevels+1, 123, 456
	s.Stats = domain.Stats{EnemiesKilled: 7, AttacksMade: 8, HitsTaken: 9, TilesMoved: 10, FoodUsed: 11, ElixirsUsed: 12, ScrollsRead: 13}
	want := []string{"456", "21 / 21", "7", "123", "8", "9", "10", "11", "12", "13"}
	for i, stat := range runSummaryStats(s) {
		if stat.value != want[i] {
			t.Fatalf("%s: got %s, want %s", stat.label, stat.value, want[i])
		}
	}
	if runSummaryStats(nil)[0].value != "0" {
		t.Fatal("missing session has nonzero score")
	}
}
