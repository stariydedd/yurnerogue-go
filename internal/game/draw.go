package game

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

// Draw рисует кадр в логическую поверхность; на экран её растягивает
// DrawFinalScreen.
func (g *Game) Draw(screen *ebiten.Image) {
	// DrawFinalScreen (surface.go) показывает логическую поверхность, а не offscreen
	// Ebitengine. Ebitengine вызывает его, только пока offscreen меняется, поэтому
	// новый кадр трогает offscreen одним пикселем, чтобы его показали, а неизменный
	// кадр его не трогает и пропускается целиком. Изменённый размер окна или
	// переход в полноэкранный режим тоже нужно показать, даже если кадр тот же:
	// экран устройства был создан заново.
	surface := g.ensureSurface()
	size := screen.Bounds().Size()
	resized := size != g.shownSize
	g.shownSize = size
	reused := g.reuseFrame(surface)
	if reused && !resized {
		return
	}
	if !reused {
		g.drawFrame(surface)
	}
	if g.shownMark == nil {
		g.shownMark = ebiten.NewImage(1, 1)
	}
	screen.DrawImage(g.shownMark, nil)
}

// drawFrame рисует кадр текущего экрана в логическую поверхность.
func (g *Game) drawFrame(surface *ebiten.Image) {
	r := g.renderer
	g.frames++
	g.drawScreen(surface)

	page, menu := menuPage(g.state)
	if menu {
		if g.state == StatePauseMenu {
			settings := g.audio.Settings()
			r.DrawPauseMenu(surface, g.pauseSelected, settings.Music, settings.Effects)
		} else if g.state == StateQuitDialog {
			r.DrawQuitDialog(surface, g.quitSelected)
		} else {
			selected := g.menuSelected
			if g.state == StateWelcome || g.state == StateDeath || g.state == StateWin {
				selected = g.runMenuSelected
			}
			r.DrawMenuButtons(surface, page, selected)
		}
		if g.state == StateMainMenu {
			settings := g.audio.Settings()
			r.DrawAudioControls(surface, settings.Music, settings.Effects, g.menuSelected)
		}
	} else if g.controls != nil {
		var player *domain.Person
		if g.session != nil {
			player = g.session.Player
		}
		g.controls.Draw(surface, r, g.pressedControls(), selectLabel(g.state), runControlVisible(g.state), player)
	}
}

// pressedControls: контролы, которые сейчас удерживаются (для подсветки).
func (g *Game) pressedControls() map[string]bool {
	held := map[string]bool{}
	if g.state == StatePlaying && g.pendingRun {
		held[render.CtrlRun] = true
	}
	if g.touch == nil {
		return held
	}
	for _, control := range g.touch.pressed {
		held[control] = true
	}
	return held
}

// drawScreen рисует содержимое текущего экрана.
func (g *Game) drawScreen(screen *ebiten.Image) {
	r := g.renderer

	switch g.state {
	case StateStarting:
		r.DrawEndScreen(screen, "CONNECTING", "Starting game...")
	case StateMainMenu:
		r.DrawMainMenu(screen, g.menuSelected, g.menuMessage)
	case StateNameEntry:
		r.DrawNameEntry(screen, g.nameInput, g.submitStatus)
	case StateWelcome:
		r.DrawWelcome(screen)
	case StatePlaying, StatePauseMenu:
		r.DrawWorld(screen, g.session)
		r.DrawHUD(screen, g.session)
	case StateItemMenu:
		r.DrawWorld(screen, g.session)
		r.DrawHUD(screen, g.session)
		r.DrawItemMenu(screen, g.itemMenuItems, g.itemMenuSelected)
	case StateQuitDialog:
		r.DrawWorld(screen, g.session)
		r.DrawHUD(screen, g.session)
	case StateLeaderboard:
		r.DrawLeaderboard(screen, g.leaderboard, g.leaderboardLoading, g.leaderboardSource)
	case StateHelp:
		r.DrawHelp(screen, render.MenuHelp, g.helpScroll)
	case StateGlossary:
		r.DrawHelp(screen, render.MenuGlossary, g.helpScroll)
	case StateDeath, StateWin:
		r.DrawRunSummary(screen, g.session, g.state == StateWin, g.submitStatus, g.placement)
	}
}

// frameMemo описывает последний кадр, нарисованный на поверхность.
type frameMemo struct {
	drawn     bool
	surface   *ebiten.Image
	session   *domain.Session
	state     State
	key       uint64
	anim      int
	animating bool
}

// reuseFrame сообщает, есть ли этот кадр уже на поверхности. Во время игры
// картинка меняется только на кадре анимации, пока движутся персонажи или
// отметки, или когда меняются ход, HUD или контролы; между этим поверхность
// показывается заново, а не перерисовывается, и неподвижная игра почти
// не нагружает машину. После конца движения рисуется ещё один кадр, чтобы
// показать последнюю позицию. Остальные экраны рисуются каждый кадр, как раньше.
func (g *Game) reuseFrame(surface *ebiten.Image) bool {
	r := g.renderer
	if r == nil || g.session == nil || g.state != StatePlaying {
		g.frame = frameMemo{}
		return false
	}
	h := sceneHash(r.SceneKey(g.session))
	h.flag(g.pendingRun)
	h.text(g.queuedAction)
	for _, control := range g.pressedControlNames() {
		h.text(control)
	}
	now := frameMemo{drawn: true, surface: surface, session: g.session, state: g.state,
		key: uint64(h), anim: r.AnimTick(), animating: r.Animating()}
	reuse := g.frame.drawn && !g.frame.animating && !now.animating &&
		g.frame.surface == now.surface && g.frame.session == now.session && g.frame.state == now.state &&
		g.frame.key == now.key && g.frame.anim == now.anim
	g.frame = now
	return reuse
}

// sceneHash подмешивает собственные входные данные кадра игры в ключ сцены рендерера.
type sceneHash uint64

func (h *sceneHash) flag(v bool) {
	*h = (*h ^ 2) * 1099511628211
	if v {
		*h = (*h ^ 1) * 1099511628211
	}
}

func (h *sceneHash) text(s string) {
	*h = (*h ^ sceneHash(len(s))) * 1099511628211
	for i := 0; i < len(s); i++ {
		*h = (*h ^ sceneHash(s[i])) * 1099511628211
	}
}

// pressedControlNames перечисляет удерживаемые контролы в стабильном порядке.
func (g *Game) pressedControlNames() []string {
	held := g.pressedControls()
	names := make([]string, 0, len(held))
	for name := range held {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
