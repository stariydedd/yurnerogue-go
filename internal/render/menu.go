package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type MenuPage int

const (
	MenuHome MenuPage = iota
	MenuName
	MenuBack
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
		active := menuButtonActive(page, button, i, selected)
		r.uiSlot(screen, button.Bounds, active)
		y := float64(button.Bounds.Min.Y+button.Bounds.Dy()/2) - 10
		r.TextCentered(screen, button.Label, r.Fonts.Menu, y, uiText)
	}
}

func menuButtonActive(page MenuPage, button MenuButton, index, selected int) bool {
	return button.Action == "start" || page == MenuHome && index == selected
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
