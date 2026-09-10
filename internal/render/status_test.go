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
		lines := r.statusLines(status, fonts.Small)
		if strings.Join(lines, " ") != status {
			t.Fatal("status text lost during wrapping")
		}
		for _, line := range lines {
			if TextWidth(line, fonts.Small) > float64(r.Layout.ScreenW-40) {
				t.Fatal("status clipped on mobile")
			}
		}
	}
}
