package render

import (
	"image"
	"testing"
)

func TestSelectCaptionFitsTouchButton(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Fatal(err)
	}
	controls := NewControls(TouchLayout(390, 844))
	width := float64(controls.targets[CtrlSelect].Dx() - 8)
	if fitLabel("SELECT", fonts.Small, width) != "SELECT" {
		t.Fatal("SELECT caption is clipped")
	}
}

func TestMenuAndHelpButtonsHaveMatchingEdges(t *testing.T) {
	for _, l := range []Layout{DesktopLayout(), TouchLayout(390, 600), TouchLayout(390, 844)} {
		targets := HUDTargets(l)
		if l.Touch {
			targets = NewControls(l).targets
		}
		menu, help := targets[CtrlMenu], targets[CtrlSelect]
		if menu.Min.X != help.Min.X || menu.Max.X != help.Max.X || menu.Dx() != 72 {
			t.Fatal("MENU must be widened to match HELP")
		}
		for _, x := range []int{menu.Min.X, menu.Max.X - 1} {
			if controlAt(targets, x, menu.Min.Y+menu.Dy()/2) != CtrlMenu {
				t.Fatal("widened MENU edges are not clickable")
			}
		}
	}
}

func TestControlsFitAndHitWithoutOverlap(t *testing.T) {
	for _, viewport := range []image.Point{{320, 568}, {360, 640}, {390, 844}, {480, 960}, {768, 1024}, {844, 390}} {
		l := TouchLayout(viewport.X, viewport.Y)
		c := NewControls(l)
		if len(c.targets) != 12 {
			t.Fatalf("missing controls: %d", len(c.targets))
		}
		for name, rect := range c.targets {
			if !rect.In(c.Panel) {
				t.Fatalf("%s outside panel at %v: %v", name, viewport, rect)
			}
			if rect.Dx() < 44 || rect.Dy() < 44 {
				t.Fatalf("target too small: %s", name)
			}
			for other, bounds := range c.targets {
				if name != other && rect.Overlaps(bounds) {
					t.Fatalf("%s overlaps %s", name, other)
				}
			}
			for y := rect.Min.Y; y < rect.Max.Y; y++ {
				for x := rect.Min.X; x < rect.Max.X; x++ {
					if got := c.ControlAt(x, y); got != name {
						t.Fatalf("%s target hits %s at %d,%d", name, got, x, y)
					}
				}
			}
		}
	}
}

func TestControlAtIgnoresEmptySpaceAndHubWaits(t *testing.T) {
	l := TouchLayout(390, 844)
	c := NewControls(l)
	if c.targets[CtrlWait].Dx() != 44 || c.targets[CtrlWait].Dy() != 44 {
		t.Fatal("wait should be smaller than the directional buttons")
	}
	for y := c.hub.Min.Y; y < c.hub.Max.Y; y++ {
		for x := c.hub.Min.X; x < c.hub.Max.X; x++ {
			if !image.Pt(x, y).In(c.targets[CtrlWait]) && c.ControlAt(x, y) != "" {
				t.Fatal("the margin around wait must not trigger a turn")
			}
		}
	}
	if got := c.ControlAt(boxCenter(c.hub).X, boxCenter(c.hub).Y); got != CtrlWait {
		t.Fatalf("D-pad centre hits %q, want WAIT", got)
	}
	for _, p := range []image.Point{{240, 10}, {240, l.GridH + 10}, {205, l.ControlsTop() + 110}, {240, l.ScreenH - 5}} {
		if got := c.ControlAt(p.X, p.Y); got != "" {
			t.Fatalf("empty point %v hits %q", p, got)
		}
	}
	if got := c.ControlAt(248, l.ControlsTop()+52); got != CtrlRun {
		t.Fatalf("separate RUN target hits %q", got)
	}
}

func TestHUDTargetsMatchDesktopSlots(t *testing.T) {
	l := DesktopLayout()
	targets := HUDTargets(l)
	if len(targets) != 6 {
		t.Fatalf("missing HUD targets: %d", len(targets))
	}
	panel := image.Rect(0, l.GridH, l.ScreenW, l.ScreenH)
	h := desktopHUDGeometry(l)
	for _, name := range []string{CtrlStrike, CtrlGuard, CtrlFood, CtrlElixir} {
		if targets[name].Min.Y != h.portrait.Min.Y {
			t.Fatalf("%s must align with the portrait top", name)
		}
		badge := h.keyBadge(targets[name])
		if badge.Min.Y <= targets[name].Max.Y || badge.Max.Y != h.portrait.Max.Y || badge.Overlaps(h.effects) {
			t.Fatalf("%s key badge must share the lower baseline without overlapping icons or effects: %v", name, badge)
		}
	}
	if h.effects.Min.X != h.status.Min.X || h.effects.Max.Y > h.log.Min.Y {
		t.Fatal("effects must stay in the right status area above the log")
	}
	if h.portrait.Min.Y-l.GridTop() != l.ControlsTop()-h.portrait.Max.Y || l.PanelH != 128 {
		t.Fatal("desktop HUD must keep equal portrait padding above and below")
	}
	if _, exists := targets[CtrlWait]; exists {
		t.Fatal("desktop must not show a wait button")
	}
	if h.status.Min.Y != h.health.Min.Y || targets[CtrlMenu].Min.Y != h.health.Min.Y || h.log.Max.Y != h.portrait.Max.Y || targets[CtrlSelect].Max.Y != h.portrait.Max.Y {
		t.Fatal("desktop HUD groups must share top and bottom alignment")
	}
	for name, rect := range targets {
		if !rect.In(panel) {
			t.Fatalf("%s outside HUD", name)
		}
		p := boxCenter(rect)
		if got := HUDControlAt(l, p.X, p.Y); got != name {
			t.Fatalf("%s hits %s", name, got)
		}
		for other, bounds := range targets {
			if name != other && rect.Overlaps(bounds) {
				t.Fatalf("%s overlaps %s", name, other)
			}
		}
	}
	if HUDControlAt(l, 500, 100) != "" {
		t.Fatal("world must not contain HUD targets")
	}
	if len(HUDTargets(TouchLayout(480, 960))) != 0 {
		t.Fatal("touch HUD must not contain desktop targets")
	}
}

func TestTouchAbilityColumnOrder(t *testing.T) {
	for _, size := range []image.Point{{320, 568}, {390, 844}, {430, 932}} {
		l := TouchLayout(size.X, size.Y)
		c := NewControls(l)
		if c.targets[CtrlStrike].Min.X != c.targets[CtrlGuard].Min.X || c.targets[CtrlStrike].Min.Y >= c.targets[CtrlGuard].Min.Y {
			t.Fatal("strike must be above defense")
		}
		if c.targets[CtrlFood].Min.X != c.targets[CtrlElixir].Min.X || c.targets[CtrlFood].Min.Y >= c.targets[CtrlElixir].Min.Y {
			t.Fatal("food must be above elixir")
		}
		for _, name := range []string{CtrlStrike, CtrlGuard} {
			if c.targets[name].Empty() {
				t.Fatalf("missing ability %s", name)
			}
		}
	}
}
