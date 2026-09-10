package render

import (
	"image"
	"testing"
)

func TestMenuButtonsFitAndMatchHitTargets(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844), TouchLayout(430, 932), TouchLayout(844, 390)} {
		for _, page := range []MenuPage{MenuHome, MenuName, MenuBack} {
			buttons := MenuButtons(l, page)
			for i, button := range buttons {
				if !button.Bounds.In(image.Rect(0, 0, l.ScreenW, l.ScreenH)) || button.Bounds.Dy() < 60 {
					t.Fatalf("button outside screen or too small: %+v", button)
				}
				if TextWidth(button.Label, fonts.Menu) > float64(button.Bounds.Dx()-24) {
					t.Fatalf("label does not fit: %s", button.Label)
				}
				p := boxCenter(button.Bounds)
				if MenuActionAt(l, page, p.X, p.Y) != button.Action {
					t.Fatal("hit target differs from drawn button")
				}
				for _, other := range buttons[i+1:] {
					if button.Bounds.Overlaps(other.Bounds) {
						t.Fatal("buttons overlap")
					}
				}
			}
			if MenuActionAt(l, page, 5, l.ScreenH-5) != "" {
				t.Fatal("empty background is active")
			}
		}
	}
}

func TestMenuSelectionUsesSameButtonsOnDesktopAndMobile(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 700)} {
		buttons := MenuButtons(l, MenuHome)
		for selected := range MainMenuOptions {
			for i, button := range buttons {
				if button.Action != MainMenuOptions[i].Key || button.Label != MainMenuOptions[i].Label {
					t.Fatal("pointer and keyboard menu options differ")
				}
				if menuButtonActive(MenuHome, button, i, selected) != (i == selected) {
					t.Fatal("wrong keyboard selection highlighted")
				}
			}
		}
	}
}
