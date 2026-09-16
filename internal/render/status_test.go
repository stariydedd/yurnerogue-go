package render

import (
	"strings"
	"testing"
)

func TestNetworkStatusFitsMobileScreen(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	r := &Renderer{Layout: TouchLayout(390, 700), Fonts: fonts}
	for _, status := range []string{
		"Score not submitted: leaderboard unavailable for this game.",
		"Connection timed out. Press PLAY to retry.",
		"Server rejected the start. Reload and retry.",
		"Invalid server response. Reload and retry.",
	} {
		lines := r.statusLines(status, r.secondaryFace())
		if strings.Join(lines, " ") != status {
			t.Fatal("status text lost during wrapping")
		}
		for _, line := range lines {
			if TextWidth(line, r.secondaryFace()) > float64(r.Layout.ScreenW-40) {
				t.Fatal("status clipped on mobile")
			}
		}
	}
}

func TestSecondaryCaptionsUseReadableSharedStyle(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		r := &Renderer{Layout: l, Fonts: fonts}
		face := r.secondaryFace()
		if TextWidth("M", face) <= TextWidth("M", fonts.Small) {
			t.Fatal("secondary text is still tiny")
		}
		for _, label := range []string{strings.ToUpper(Tagline), "EVERY STEP COUNTS", "EMPTY NAME = ANONYMOUS", "RUN SUMMARY / " + strings.Repeat("я", 16), "GLOBAL", "SERVER UNAVAILABLE"} {
			lines := r.statusLines(label, face)
			if len(lines) != 1 || TextWidth(lines[0], face) > float64(l.ScreenW-40) {
				t.Fatalf("caption no longer fits its header: %s", label)
			}
		}
		for _, status := range []string{"Connection timed out. Press PLAY to retry.", "Server rejected the start. Reload and retry.", "Cannot reach leaderboard. Press PLAY to retry."} {
			bottom := float64(NameEntryBounds(l).Max.Y+58) + float64(len(r.statusLines(status, face)))*(TextWidth("M", face)+6)
			if bottom > float64(MenuButtons(l, MenuName)[0].Bounds.Min.Y) {
				t.Fatal("name status overlaps PLAY")
			}
		}
	}
}
