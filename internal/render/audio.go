package render

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

func AudioTargets(l Layout) map[string]image.Rectangle {
	x := l.ScreenW/2 - 180
	y := l.ScreenH - 120
	return map[string]image.Rectangle{
		"music":   image.Rect(x, y, x+172, y+38),
		"effects": image.Rect(x+188, y, x+360, y+38),
	}
}

func AudioControlAt(l Layout, x, y int) string { return controlAt(AudioTargets(l), x, y) }

func (r *Renderer) DrawAudioControls(screen *ebiten.Image, music, effects int) {
	for name, box := range AudioTargets(r.Layout) {
		value, label := effects, "SFX"
		if name == "music" {
			value, label = music, "MUSIC"
		}
		r.uiSlot(screen, box, false)
		label = fmt.Sprintf("%s %d%%", label, value)
		if !r.Layout.Touch {
			key := "V"
			if name == "music" {
				key = "M"
			}
			label = "[" + key + "] " + label
		}
		r.slotLabel(screen, label, box, uiText)
	}
}
