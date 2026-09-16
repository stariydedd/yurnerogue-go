package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

func TestPauseSlidersKeyboardPointerAndDrag(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		g.audio = &sound.Engine{}
		g.startNewGame()
		session, stats, actions := g.session, g.session.Stats, g.session.Actions()
		g.runTicket = "preserved"
		g.HandleKey(ebiten.KeyQ)
		g.HandleKey(ebiten.KeyRight) // RESUME is not a slider.
		if g.audio.Settings() != (sound.Settings{}) {
			t.Fatal("non-slider selection changed volume")
		}
		g.HandleKey(ebiten.KeyDown)
		for _, key := range []ebiten.Key{ebiten.KeyD, ebiten.KeyRight, ebiten.KeyRight} {
			g.HandleKey(key)
		}
		g.HandleKey(ebiten.KeyA)
		g.HandleKey(ebiten.KeyLeft)
		if g.audio.Settings() != (sound.Settings{Music: 5}) {
			t.Fatal("keyboard did not change only music in 5% steps")
		}
		g.HandleKey(ebiten.KeyDown)
		g.HandleKey(ebiten.KeyD)
		if g.audio.Settings() != (sound.Settings{Music: 5, Effects: 5}) {
			t.Fatal("keyboard changed the wrong channel")
		}
		for _, button := range render.MenuButtons(l, render.MenuPause) {
			if button.Action != "music" && button.Action != "effects" {
				continue
			}
			track := button.Bounds.Inset(8)
			for _, percent := range []int{0, 25, 50, 75, 100} {
				x := track.Min.X + (track.Dx()*percent+50)/100
				g.handleMenuPointer(x, track.Min.Y+10)
				settings := g.audio.Settings()
				got := settings.Effects
				if button.Action == "music" {
					got = settings.Music
				}
				if got != percent {
					t.Fatalf("pointer: got %d, want %d", got, percent)
				}
			}
			g.HandleKey(ebiten.KeyRight)
			g.audioDrag = button.Action
			g.moveAudioSlider(track.Min.X - 100)
			g.HandleKey(ebiten.KeyLeft)
			settings := g.audio.Settings()
			got := settings.Effects
			if button.Action == "music" {
				got = settings.Music
			}
			if got != 0 {
				t.Fatal("drag or keyboard did not clamp at zero")
			}
			g.moveAudioSlider(track.Max.X + 100)
			g.HandleKey(ebiten.KeyRight)
			settings = g.audio.Settings()
			got = settings.Effects
			if button.Action == "music" {
				got = settings.Music
			}
			if got != 100 {
				t.Fatal("drag or keyboard wrapped past 100")
			}
		}
		if g.state != StatePauseMenu || g.session != session || g.session.Stats != stats || g.session.Actions() != actions || g.runTicket != "preserved" {
			t.Fatal("slider changed the run")
		}
	}
}

func TestMainMenuAudioSliders(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		g.audio = &sound.Engine{}
		for range 3 {
			g.HandleKey(ebiten.KeyDown)
		}
		if render.MainMenuOptions[g.menuSelected].Key != "music" {
			t.Fatal("keyboard cannot select music")
		}
		g.HandleKey(ebiten.KeyD)
		g.HandleKey(ebiten.KeyRight)
		g.HandleKey(ebiten.KeyA)
		if g.audio.Settings().Music != 5 {
			t.Fatal("main menu music keyboard adjustment failed")
		}
		g.HandleKey(ebiten.KeyDown)
		g.HandleKey(ebiten.KeyRight)
		if g.audio.Settings() != (sound.Settings{Music: 5, Effects: 5}) {
			t.Fatal("main menu slider changed the wrong channel")
		}
		for name, bounds := range render.AudioTargets(l) {
			track := bounds.Inset(8)
			x := track.Min.X + (track.Dx()*75+50)/100
			g.handleMenuPointer(x, track.Min.Y+10)
			if render.MainMenuOptions[g.menuSelected].Key != name {
				t.Fatal("pointer did not focus the slider")
			}
			g.HandleKey(ebiten.KeyLeft)
			got := g.audio.Settings().Effects
			if name == "music" {
				got = g.audio.Settings().Music
			}
			if got != 70 {
				t.Fatalf("click plus keyboard: got %d, want 70", got)
			}
			g.audioDrag = name
			g.moveAudioSlider(track.Max.X + 100)
			got = g.audio.Settings().Effects
			if name == "music" {
				got = g.audio.Settings().Music
			}
			if got != 100 {
				t.Fatal("main menu drag did not clamp")
			}
		}
		if g.state != StateMainMenu || g.session != nil || g.startResults != nil {
			t.Fatal("volume input started a game")
		}
		g.menuSelected = len(render.MainMenuOptions) - 1
		g.HandleKey(ebiten.KeyDown)
		if g.menuSelected != 0 {
			t.Fatal("menu navigation did not wrap back to PLAY")
		}
	}
}
