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

func TestControlsFitAndHitWithoutOverlap(t *testing.T) {
	for _, viewport := range []image.Point{{320, 568}, {360, 640}, {390, 844}, {480, 960}, {768, 1024}, {844, 390}} {
		l := TouchLayout(viewport.X, viewport.Y)
		c := NewControls(l)
		if len(c.targets) != 11 {
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

func TestControlAtIgnoresHubAndEmptySpace(t *testing.T) {
	l := TouchLayout(390, 844)
	c := NewControls(l)
	for _, p := range []image.Point{{240, 10}, {240, l.GridH + 10}, boxCenter(c.hub), {205, l.ControlsTop() + 110}, {240, l.ScreenH - 5}} {
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
