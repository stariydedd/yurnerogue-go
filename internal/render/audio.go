package render

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func AudioTargets(l Layout) map[string]image.Rectangle {
	w := min(172, l.ScreenW-48)
	x := (l.ScreenW - w) / 2
	y := l.ScreenH - 184
	return map[string]image.Rectangle{
		"music":   image.Rect(x, y, x+w, y+64),
		"effects": image.Rect(x, y+80, x+w, y+144),
	}
}

func AudioControlAt(l Layout, x, y int) string { return controlAt(AudioTargets(l), x, y) }

func (r *Renderer) DrawAudioControls(screen *ebiten.Image, music, effects int, selected ...int) {
	for i, button := range MenuButtons(r.Layout, MenuHome) {
		name := button.Action
		if name != "music" && name != "effects" {
			continue
		}
		value, label := effects, "SFX"
		if name == "music" {
			value, label = music, "MUSIC"
		}
		label = fmt.Sprintf("%s %d%%", label, value)
		button.Label = label
		r.drawVolumeButton(screen, button, value, len(selected) > 0 && selected[0] == i, r.Fonts.UI)
	}
}

func (r *Renderer) drawVolumeButton(screen *ebiten.Image, button MenuButton, volume int, active bool, face text.Face) {
	r.uiSlot(screen, button.Bounds, active)
	fillBox(screen, button.Bounds.Inset(8), uiRecess)
	fillBox(screen, PauseVolumeFill(button.Bounds, volume), uiEdge)
	x := float64(button.Bounds.Min.X+button.Bounds.Dx()/2) - TextWidth(button.Label, face)/2
	y := float64(button.Bounds.Min.Y+button.Bounds.Dy()/2) - TextWidth("M", face)/2 - 1
	r.Text(screen, button.Label, face, x+1, y+1, uiInk)
	r.Text(screen, button.Label, face, x, y, uiText)
}
