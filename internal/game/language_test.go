package game

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

func TestLanguageFlagsOnlyRespondToHomePointers(t *testing.T) {
	config := t.TempDir()
	t.Setenv("APPDATA", config)
	t.Setenv("XDG_CONFIG_HOME", config)
	for _, l := range []render.Layout{render.DesktopLayout(), render.TouchLayout(390, 600)} {
		g := New(&render.Renderer{Layout: l})
		g.playerName = "PLAY"
		for language, box := range render.LanguageTargets(l) {
			g.handleMenuPointer(box.Min.X+20, box.Min.Y+20)
			if g.renderer.Layout.Language != language || locale.Load() != language || g.state != StateMainMenu || g.playerName != "PLAY" {
				t.Fatal("language click changed gameplay state or was not saved")
			}
			for i := 0; i < len(render.MainMenuOptions); i++ {
				g.HandleKey(ebiten.KeyDown)
			}
			if g.renderer.Layout.Language != language || g.menuSelected != 0 {
				t.Fatal("flags must not enter keyboard menu navigation")
			}
		}
		g.state = StateHelp
		g.renderer.Layout.Language = locale.English
		box := render.LanguageTargets(l)[locale.Russian]
		g.handleMenuPointer(box.Min.X+20, box.Min.Y+20)
		if g.renderer.Layout.Language != locale.English {
			t.Fatal("hidden flag was clickable")
		}
	}
}
