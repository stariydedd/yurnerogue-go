package render

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type MenuPage int

const (
	MenuHome MenuPage = iota
	MenuName
	MenuBack
	MenuPause
)

type MenuButton struct {
	Label, Action string
	Bounds        image.Rectangle
}

// Shared by drawing and hit testing; no invisible gameplay targets on menu pages.
func MenuButtons(l Layout, page MenuPage) []MenuButton {
	width := min(360, l.ScreenW-48)
	left := (l.ScreenW - width) / 2
	right := left + width
	if page == MenuPause {
		top := l.ScreenH/2 - 130
		return []MenuButton{
			{"RESUME", "resume", image.Rect(left, top, right, top+64)},
			{"MUSIC", "music", image.Rect(left, top+80, right, top+144)},
			{"SFX", "effects", image.Rect(left, top+160, right, top+224)},
			{"MAIN MENU", "quit", image.Rect(left, top+240, right, top+304)},
		}
	}
	if page == MenuHome {
		top := l.ScreenH/2 + 8
		return []MenuButton{
			{"PLAY", "new", image.Rect(left, top, right, top+64)},
			{"LEADERBOARD", "scoreboard", image.Rect(left, top+82, right, top+146)},
			{"HELP", "help", image.Rect(left, top+164, right, top+228)},
		}
	}
	back := MenuButton{"BACK", "back", image.Rect(left, l.ScreenH-88, right, l.ScreenH-24)}
	if page == MenuName {
		return []MenuButton{
			{"PLAY", "start", image.Rect(left, l.ScreenH-170, right, l.ScreenH-106)}, back,
		}
	}
	return []MenuButton{back}
}

func MenuActionAt(l Layout, page MenuPage, x, y int) string {
	for _, button := range MenuButtons(l, page) {
		if image.Pt(x, y).In(button.Bounds) {
			return button.Action
		}
	}
	return ""
}

func (r *Renderer) DrawMenuButtons(screen *ebiten.Image, page MenuPage, selected int) {
	for i, button := range MenuButtons(r.Layout, page) {
		r.drawMenuButton(screen, button, menuButtonActive(page, button, i, selected))
	}
}

func (r *Renderer) drawMenuButton(screen *ebiten.Image, button MenuButton, active bool) {
	r.uiSlot(screen, button.Bounds, active)
	y := float64(button.Bounds.Min.Y+button.Bounds.Dy()/2) - 10
	r.TextCentered(screen, button.Label, r.Fonts.Menu, y, uiText)
}

func pauseButtonLabel(button MenuButton, music, effects int) string {
	switch button.Action {
	case "music":
		return fmt.Sprintf("MUSIC %d%%", music)
	case "effects":
		return fmt.Sprintf("SFX %d%%", effects)
	}
	return button.Label
}

func (r *Renderer) DrawPauseMenu(screen *ebiten.Image, selected, music, effects int) {
	r.dimScreen(screen)
	buttons := MenuButtons(r.Layout, MenuPause)
	top := float64(buttons[0].Bounds.Min.Y)
	r.TextCentered(screen, "MENU", r.Fonts.Menu, top-70, uiAccent)
	for i, button := range buttons {
		button.Label = pauseButtonLabel(button, music, effects)
		r.drawMenuButton(screen, button, i == selected)
	}
	hint := "Volume: tap to cycle 0-100%"
	if !r.Layout.Touch {
		hint = "M / V: volume   Esc: resume"
	}
	r.TextCentered(screen, hint, r.Fonts.Small, top+328, uiMuted)
}

func menuButtonActive(page MenuPage, button MenuButton, index, selected int) bool {
	return button.Action == "start" || (page == MenuHome || page == MenuPause) && index == selected
}

func (r *Renderer) drawMenuHeader(screen *ebiten.Image, message string) {
	h := r.Layout.ScreenH
	r.backdrop(screen)
	y := float64(h/2 - 270)
	r.TextCentered(screen, "YurneROGUE", r.Fonts.Title, y, uiAccent)
	r.TextCentered(screen, Tagline, r.Fonts.Small, y+62, uiMuted)
	r.drawHeroBanner(screen, y+110, 3)
	if message != "" {
		r.TextCentered(screen, fitLabel(message, r.Fonts.Small, float64(r.Layout.ScreenW-40)), r.Fonts.Small, float64(h-56), uiMuted)
	}
}
