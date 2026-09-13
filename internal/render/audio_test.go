package render

import (
	"image"
	"testing"
)

func TestAudioControlsFitWithoutCoveringMenu(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		for name, box := range AudioTargets(l) {
			if !box.In(image.Rect(0, 0, l.ScreenW, l.ScreenH)) {
				t.Fatal("volume button outside screen")
			}
			p := boxCenter(box)
			if AudioControlAt(l, p.X, p.Y) != name {
				t.Fatal("volume hit box mismatch")
			}
			for _, button := range MenuButtons(l, MenuHome) {
				if box.Overlaps(button.Bounds) {
					t.Fatal("volume overlaps menu")
				}
			}
		}
	}
}
