package render

import (
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type MenuPage int

const (
	MenuHome MenuPage = iota
	MenuName
	MenuBack
	MenuPause
	MenuWelcome
	MenuResults
	MenuQuit
	MenuHelp
	MenuLeaderboard
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
	if page == MenuQuit {
		top := QuitDialogBounds(l).Min.Y + 132
		buttons := make([]MenuButton, len(QuitOptions))
		for i, option := range QuitOptions {
			buttons[i] = MenuButton{option.Label, option.Key, image.Rect(left, top+i*80, right, top+i*80+64)}
		}
		return buttons
	}
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
		top := homeMenuTop(l)
		audio := AudioTargets(l)
		return []MenuButton{
			{"PLAY", "new", image.Rect(left, top, right, top+64)},
			{"LEADERBOARD", "scoreboard", image.Rect(left, top+82, right, top+146)},
			{"HELP", "help", image.Rect(left, top+164, right, top+228)},
			{"MUSIC", "music", audio["music"]},
			{"SFX", "effects", audio["effects"]},
		}
	}
	back := MenuButton{"BACK", "back", image.Rect(left, l.ScreenH-88, right, l.ScreenH-24)}
	if page == MenuHelp && !l.Touch {
		top := HelpViewBounds(l).Max.Y + 20
		back.Bounds = image.Rect(left, top, right, top+64)
	}
	if page == MenuLeaderboard && leaderboardDetailed(l) {
		top := LeaderboardViewBounds(l).Max.Y + 24
		back.Bounds = image.Rect(left, top, right, top+64)
	}
	if page == MenuWelcome || page == MenuResults {
		label, action := "CONTINUE", "continue"
		if page == MenuResults {
			label, action = "PLAY AGAIN", "again"
		}
		back.Label = "MAIN MENU"
		return []MenuButton{{label, action, image.Rect(left, l.ScreenH-170, right, l.ScreenH-106)}, back}
	}
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
		if page == MenuHome && (button.Action == "music" || button.Action == "effects") {
			continue // DrawAudioControls supplies the current percentage.
		}
		r.drawMenuButton(screen, button, menuButtonActive(page, button, i, selected))
	}
}

func (r *Renderer) drawMenuButton(screen *ebiten.Image, button MenuButton, active bool) {
	r.uiSlot(screen, button.Bounds, active)
	y := float64(button.Bounds.Min.Y+button.Bounds.Dy()/2) - 10
	x := float64(button.Bounds.Min.X+button.Bounds.Dx()/2) - TextWidth(button.Label, r.Fonts.Menu)/2
	r.Text(screen, button.Label, r.Fonts.Menu, x, y, uiText)
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
	center := r.Layout.ScreenW / 2
	r.stonePanel(screen, image.Rect(center-120, int(top)-104, center+120, int(top)-28))
	r.titleWithShadow(screen, "MENU", top-88)
	for i, button := range buttons {
		button.Label = pauseButtonLabel(button, music, effects)
		if button.Action == "music" || button.Action == "effects" {
			volume := effects
			if button.Action == "music" {
				volume = music
			}
			r.drawVolumeButton(screen, button, volume, i == selected, r.Fonts.Menu)
		} else {
			r.drawMenuButton(screen, button, i == selected)
		}
	}
}

func PauseVolumeFill(bounds image.Rectangle, volume int) image.Rectangle {
	fill := bounds.Inset(8)
	fill.Max.X = fill.Min.X + fill.Dx()*max(0, min(100, volume))/100
	return fill
}

func PauseVolumeAt(bounds image.Rectangle, x int) int {
	track := bounds.Inset(8)
	return max(0, min(100, ((x-track.Min.X)*100+track.Dx()/2)/track.Dx()))
}

func menuButtonActive(page MenuPage, button MenuButton, index, selected int) bool {
	if (page == MenuHelp || page == MenuLeaderboard) && button.Action == "back" {
		return true
	}
	return button.Action == "start" || (page == MenuHome || page == MenuPause || page == MenuWelcome || page == MenuResults || page == MenuQuit || page == MenuHelp) && index == selected
}

func homeMenuTop(l Layout) int {
	return min(l.ScreenH/2+8, l.ScreenH-440)
}

func (r *Renderer) drawMenuHeader(screen *ebiten.Image, message string) {
	h := r.Layout.ScreenH
	r.backdrop(screen)
	y := float64(homeMenuTop(r.Layout) - 278)
	r.TextCentered(screen, "YurneROGUE", r.Fonts.Title, y, uiAccent)
	r.secondaryCaption(screen, strings.ToUpper(Tagline), y+62)
	r.drawHeroBanner(screen, y+110, 3)
	if message != "" {
		r.secondaryCaption(screen, fitLabel(message, r.secondaryFace(), float64(r.Layout.ScreenW-40)), float64(h-24))
	}
}
