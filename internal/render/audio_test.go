package render

import (
	"fmt"
	"image"
	"testing"
)

func TestAudioControlsFitWithoutCoveringMenu(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		for name, box := range AudioTargets(l) {
			label := "SFX"
			if name == "music" {
				label = "MUSIC"
			}
			for volume := 0; volume <= 100; volume++ {
				if TextWidth(fmt.Sprintf("%s %d%%", label, volume), fonts.UI) > float64(box.Dx()-16) {
					t.Fatal("volume label does not fit")
				}
			}
			if !box.In(image.Rect(0, 0, l.ScreenW, l.ScreenH)) {
				t.Fatal("volume button outside screen")
			}
			p := boxCenter(box)
			if AudioControlAt(l, p.X, p.Y) != name {
				t.Fatal("volume hit box mismatch")
			}
			for _, button := range MenuButtons(l, MenuHome) {
				if button.Action == name {
					if box != button.Bounds {
						t.Fatal("audio and keyboard menu bounds differ")
					}
					continue
				}
				if box.Overlaps(button.Bounds) {
					t.Fatal("volume overlaps menu")
				}
			}
		}
	}
}

func TestMainMenuAudioSlidersStackVertically(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		targets := AudioTargets(l)
		music, effects := targets["music"], targets["effects"]
		play := MenuButtons(l, MenuHome)[0].Bounds
		if music.Dx() >= play.Dx() || boxCenter(music).X != boxCenter(play).X || effects.Min.X != music.Min.X || effects.Max.X != music.Max.X || effects.Min.Y < music.Max.Y+16 {
			t.Fatal("audio sliders are not compact and stacked")
		}
	}
}
