package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func (g *Game) audioSliderPage() (render.MenuPage, bool) {
	switch g.state {
	case StateMainMenu:
		return render.MenuHome, true
	case StatePauseMenu:
		return render.MenuPause, true
	}
	return 0, false
}

func (g *Game) setAudioSliderAt(x, y int) bool {
	page, ok := g.audioSliderPage()
	if !ok {
		return false
	}
	for i, button := range render.MenuButtons(g.renderer.Layout, page) {
		if (button.Action == "music" || button.Action == "effects") && image.Pt(x, y).In(button.Bounds) {
			if g.state == StateMainMenu {
				g.menuSelected = i
			} else {
				g.pauseSelected = i
			}
			g.audio.SetVolume(button.Action == "music", render.PauseVolumeAt(button.Bounds, x))
			return true
		}
	}
	return false
}

func (g *Game) moveAudioSlider(x int) {
	page, ok := g.audioSliderPage()
	if !ok {
		return
	}
	for _, button := range render.MenuButtons(g.renderer.Layout, page) {
		if button.Action == g.audioDrag {
			g.audio.SetVolume(button.Action == "music", render.PauseVolumeAt(button.Bounds, x))
			return
		}
	}
}

// Capture the initiating finger/mouse until release, even outside the slider.
func (g *Game) updateAudioSlider() bool {
	page, ok := g.audioSliderPage()
	if !ok || !ebiten.IsFocused() {
		g.audioDrag = ""
		return false
	}
	if g.audioDrag != "" {
		var x, y int
		if g.audioDragID == mouseID {
			if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				g.audioDrag = ""
				return true
			}
			x, y = ebiten.CursorPosition()
		} else {
			active := false
			for _, id := range ebiten.AppendTouchIDs(nil) {
				active = active || id == g.audioDragID
			}
			if !active {
				g.audioDrag = ""
				return true
			}
			x, y = ebiten.TouchPosition(g.audioDragID)
		}
		x, _ = g.toLogical(x, y)
		g.moveAudioSlider(x)
		return true
	}
	id := mouseID
	var x, y int
	if ids := inpututil.AppendJustPressedTouchIDs(nil); len(ids) > 0 {
		id = ids[0]
		x, y = ebiten.TouchPosition(id)
	} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y = ebiten.CursorPosition()
	} else {
		return false
	}
	x, y = g.toLogical(x, y)
	if !g.setAudioSliderAt(x, y) {
		return false
	}
	g.audioDragID = id
	selected := g.pauseSelected
	if g.state == StateMainMenu {
		selected = g.menuSelected
	}
	g.audioDrag = render.MenuButtons(g.renderer.Layout, page)[selected].Action
	return true
}

func (g *Game) adjustAudioSlider(action string, key ebiten.Key) {
	if action != "music" && action != "effects" {
		return
	}
	settings := g.audio.Settings()
	volume := settings.Effects
	if action == "music" {
		volume = settings.Music
	}
	delta := 5
	if key == ebiten.KeyLeft || key == ebiten.KeyA {
		delta = -delta
	}
	g.audio.SetVolume(action == "music", volume+delta)
}
