package render

import (
	"image"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func TestEventLogTextHasEqualVerticalMargins(t *testing.T) {
	for _, frame := range []image.Rectangle{desktopHUDGeometry(DesktopLayout()).log, image.Rect(10, 104, 470, 148)} {
		for rows := 1; rows <= eventLogContentBounds(frame).Dy()/16; rows++ {
			const glyphHeight = 10
			top := eventLogTextTop(frame, rows, 16, glyphHeight)
			bottom := top + (rows-1)*16 + glyphHeight
			if difference := (top - frame.Min.Y) - (frame.Max.Y - bottom); difference < -1 || difference > 1 {
				t.Fatalf("unequal visible text margins for %d rows: %v", rows, frame)
			}
		}
	}
}

func TestDesktopEventLogFitsFourRows(t *testing.T) {
	for _, size := range []image.Point{{1280, 832}, {1920, 1080}, {3440, 1440}, {800, 600}} {
		l := DesktopLayoutForSize(size.X, size.Y)
		h := desktopHUDGeometry(l)
		if rows := eventLogContentBounds(h.log).Dy() / eventLogRowHeight; rows != 4 {
			t.Fatalf("desktop log at %v fits %d rows, want 4", size, rows)
		}
		if h.log.Min.Y < h.effects.Max.Y || h.log.Max.Y != h.portrait.Max.Y {
			t.Fatal("log must stay below effects and align with the portrait bottom")
		}
		if eventLogRowHeight != 16 || h.log.Dy() != 76 {
			t.Fatal("desktop log must provide four comfortably spaced rows")
		}
	}
}

func TestEventLogFrameHasSymmetricPadding(t *testing.T) {
	for _, frame := range []image.Rectangle{image.Rect(10, 104, 470, 148), desktopHUDGeometry(DesktopLayout()).log} {
		content := eventLogContentBounds(frame)
		if content.Min.X-frame.Min.X != frame.Max.X-content.Max.X || content.Min.Y-frame.Min.Y != frame.Max.Y-content.Max.Y {
			t.Fatal("event log padding must be symmetric")
		}
		if !content.In(frame) || content.Dy()/16 < 2 {
			t.Fatal("event log must fit at least two rows inside its frame")
		}
	}
}

func TestEventLogFitsAndTranslatesAtDrawTime(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	r := &Renderer{Fonts: fonts, Layout: DesktopLayout()}
	s := domain.NewSessionSeed(21)
	s.SetMessage("You missed the Pudge.")
	s.SetMessage("Picked up: Tango.")
	for _, language := range []locale.Language{locale.English, locale.Russian} {
		r.Layout.Language = language
		lines := r.eventLogLines(s, 380, 4)
		if len(lines) != 2 || lines[0].latest || !lines[1].latest {
			t.Fatalf("wrong history ordering: %+v", lines)
		}
		if lines[1].text != r.translateMessage("Picked up: Tango.") {
			t.Fatal("history translation is stale")
		}
		for _, width := range []int{100, 380, 452, 900} {
			for _, line := range r.eventLogLines(s, width, 2) {
				if TextWidth(line.text, fonts.Small) > float64(width) {
					t.Fatal("event text overflows")
				}
			}
		}
	}
	if got := r.eventLogLines(s, 380, 1); len(got) != 1 || !got[0].latest {
		t.Fatal("latest message must have priority")
	}
	if len(r.eventLogLines(s, 0, 4)) != 0 || len(r.eventLogLines(s, 380, 0)) != 0 {
		t.Fatal("empty region must draw nothing")
	}
}
