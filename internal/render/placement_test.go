package render

import (
	"strconv"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/locale"
)

func TestPlaceColorsMarkThePodium(t *testing.T) {
	for place, want := range map[int]any{1: placeGold, 2: placeSilver, 3: placeBronze, 4: uiText, 10: uiText, 11: uiMuted, 250: uiMuted} {
		if got := placeColor(place); got != want {
			t.Fatalf("place %d: color %v, want %v", place, got, want)
		}
	}
}

func TestSizedFontsAreMadeOnce(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	if fonts.Sized(96) != fonts.Sized(96) || fonts.Sized(96) == fonts.Sized(64) {
		t.Fatal("a size must map to one kept face")
	}
}

func TestPhoneTitleClearsHeroAndMedal(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	hero := runHero{scale: 2, x: 16, y: 20, w: 64, h: 64}
	// Every phone is laid out 480 wide, portrait or landscape; desktop is
	// always wide enough to stand the hero beside the panel.
	for _, l := range []Layout{TouchLayout(360, 740), TouchLayout(430, 932), TouchLayout(844, 390)} {
		width := l.ScreenW
		for _, language := range []locale.Language{locale.English, locale.Russian} {
			r := &Renderer{Layout: l, Fonts: fonts}
			r.Layout.Language = language
			for _, title := range []string{"VICTORY", "YOU DIED"} {
				// The place arrives after the screen opens: unknown first.
				first := r.runHeader(title, Placement{}, hero)
				if other := r.runHeader(title, Placement{Place: 42, GoldShort: 5}, hero); other.face != first.face {
					t.Errorf("%s %q: the title changes size outside the top 10", language, title)
				}
				for place := 0; place <= 10; place++ {
					head := r.runHeader(title, Placement{Place: place}, hero)
					if head.face != first.face {
						t.Errorf("%s %q place %d: the title changes size when the place arrives", language, title, place)
					}
					tw := TextWidth(head.title, head.face)
					if center := head.x + tw/2; center != float64(width)/2 {
						t.Errorf("%s %q place %d: title centered at %.0f, not %d", language, title, place, center, width/2)
					}
					inset := hero.x + hero.w + phoneTitleGap
					if place > 0 {
						inset = max(inset, 16+len("#"+strconv.Itoa(place))*head.numberSize+phoneTitleGap)
						if place < 10 && head.numberSize != phoneMedal {
							t.Errorf("%s %q place %d: medal %d px, want %d", language, title, place, head.numberSize, phoneMedal)
						}
					}
					if head.x < float64(inset) || head.x+tw > float64(width-inset) {
						t.Errorf("%s %q place %d: title spans %.0f..%.0f, clear room is %d..%d", language, title, place, head.x, head.x+tw, inset, width-inset)
					}
				}
			}
		}
	}
}
