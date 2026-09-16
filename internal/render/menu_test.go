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
		for _, page := range []MenuPage{MenuHome, MenuName, MenuBack, MenuPause, MenuWelcome, MenuResults, MenuQuit, MenuHelp, MenuLeaderboard} {
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

func TestPauseVolumeLabelsFitAtEverySetting(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(320, 568), TouchLayout(390, 600), TouchLayout(844, 390)} {
		for _, button := range MenuButtons(l, MenuPause) {
			for volume := 0; volume <= 100; volume += 25 {
				label := pauseButtonLabel(button, volume, volume)
				if TextWidth(label, fonts.Menu) > float64(button.Bounds.Dx()-24) {
					t.Fatalf("pause label does not fit: %s", label)
				}
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

func TestQuitDialogContentFitsPanel(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(320, 568), TouchLayout(390, 600), TouchLayout(844, 390)} {
		box := QuitDialogBounds(l)
		if !box.In(image.Rect(0, 0, l.ScreenW, l.ScreenH)) {
			t.Fatal("confirmation outside screen")
		}
		for i, button := range MenuButtons(l, MenuQuit) {
			if !button.Bounds.In(box.Inset(16)) || button.Action != QuitOptions[i].Key {
				t.Fatal("confirmation button outside panel or wrong action")
			}
			if menuButtonActive(MenuQuit, button, i, 1) != (i == 1) {
				t.Fatal("cancel selection is not highlighted")
			}
		}
		if TextWidth("This run will be lost.", fonts.UI) > float64(box.Dx()-32) {
			t.Fatal("enlarged warning does not fit")
		}
	}
}

func TestPauseVolumeFillMatchesPercentage(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600)} {
		for _, button := range MenuButtons(l, MenuPause) {
			if button.Action != "music" && button.Action != "effects" {
				continue
			}
			track := button.Bounds.Inset(8)
			for _, volume := range []int{-10, 0, 1, 25, 50, 75, 99, 100, 110} {
				fill := PauseVolumeFill(button.Bounds, volume)
				if fill.Dx() != track.Dx()*max(0, min(100, volume))/100 || fill.Min != track.Min || fill.Max.Y != track.Max.Y {
					t.Fatal("volume fill is not proportional")
				}
			}
			if PauseVolumeAt(button.Bounds, track.Min.X-100) != 0 || PauseVolumeAt(button.Bounds, track.Max.X+100) != 100 {
				t.Fatal("pointer value outside 0-100")
			}
		}
	}
}

func TestReferenceBackButtonAlwaysHighlighted(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600)} {
		for _, page := range []MenuPage{MenuHelp, MenuLeaderboard} {
			buttons := MenuButtons(l, page)
			if len(buttons) != 1 || buttons[0].Action != "back" {
				t.Fatal("reference screen should have only BACK")
			}
			for selected := range MainMenuOptions {
				if !menuButtonActive(page, buttons[0], 0, selected) {
					t.Fatal("BACK highlight depends on previous menu selection")
				}
			}
		}
	}
}
