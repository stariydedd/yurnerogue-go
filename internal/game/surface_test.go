package game

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/locale"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

type capturedFinalScreen struct {
	ebiten.FinalScreen
	bounds  image.Rectangle
	source  *ebiten.Image
	options ebiten.DrawImageOptions
}

func TestDesktopResizePreservesGameAndPointerAlignment(t *testing.T) {
	l := render.DesktopLayout()
	l.Language = locale.Russian
	session := domain.NewSessionSeed(21)
	g := &Game{renderer: &render.Renderer{Layout: l}, session: session, state: StatePlaying, playerName: "tester"}
	for _, size := range []image.Point{{1280, 848}, {1920, 1080}, {2560, 1440}, {3440, 1440}, {3840, 2160}, {1365, 769}, {800, 600}, {1280, 848}} {
		g.Layout(size.X, size.Y)
		if g.renderer.Layout.Language != locale.Russian || g.session != session || session.Turns != 0 || g.state != StatePlaying || g.playerName != "tester" {
			t.Fatal("resizing changed game state or language")
		}
		surface := g.ensureSurface()
		if surface.Bounds().Size() != image.Pt(g.renderer.Layout.ScreenW, g.renderer.Layout.ScreenH) || g.ensureSurface() != surface {
			t.Fatal("surface does not match layout or is recreated each frame")
		}
		for _, dpr := range []float64{1, 1.25, 1.5, 2, 3} {
			final := &capturedFinalScreen{bounds: image.Rect(0, 0, int(float64(size.X)*dpr), int(float64(size.Y)*dpr))}
			g.DrawFinalScreen(final, nil, ebiten.GeoM{})
			transform := final.options.GeoM
			if final.source != surface || final.options.Filter != ebiten.FilterNearest || transform.Element(0, 0) != transform.Element(1, 1) {
				t.Fatal("desktop final pass distorts or blurs the source")
			}
			// Match real window/device rounding rather than assuming integer DPR.
			rx, ry := float64(final.bounds.Dx())/float64(size.X), float64(final.bounds.Dy())/float64(size.Y)
			for name, box := range render.HUDTargets(g.renderer.Layout) {
				point := image.Pt(box.Min.X+box.Dx()/2, box.Min.Y+box.Dy()/2)
				x, y := transform.Apply(float64(point.X), float64(point.Y))
				lx, ly := g.toLogical(int(math.Round(x/rx)), int(math.Round(y/ry)))
				if render.HUDControlAt(g.renderer.Layout, lx, ly) != name {
					t.Fatalf("DPR %v at %v: pointer missed %s", dpr, size, name)
				}
			}
			for _, button := range render.MenuButtons(g.renderer.Layout, render.MenuHome) {
				x, y := transform.Apply(float64(button.Bounds.Min.X+button.Bounds.Dx()/2), float64(button.Bounds.Min.Y+button.Bounds.Dy()/2))
				lx, ly := g.toLogical(int(math.Round(x/rx)), int(math.Round(y/ry)))
				if render.MenuActionAt(g.renderer.Layout, render.MenuHome, lx, ly) != button.Action {
					t.Fatalf("resized menu target missed: %s", button.Action)
				}
			}
		}
	}
	previous := g.renderer.Layout
	g.Layout(0, 0)
	if g.renderer.Layout != previous || g.outW <= 0 || g.outH <= 0 {
		t.Fatal("minimizing the window reset the layout")
	}
	g.surface.Deallocate()
}

func TestResizeDoesNotChangeTouchLayout(t *testing.T) {
	l := render.TouchLayout(390, 844)
	l.Language = locale.Russian
	g := New(&render.Renderer{Layout: l})
	controls := g.controls
	for _, size := range []image.Point{{390, 844}, {390, 600}, {844, 390}, {1920, 1080}} {
		g.Layout(size.X, size.Y)
		if g.renderer.Layout != l || g.controls != controls {
			t.Fatal("desktop adaptation changed mobile layout or controls")
		}
	}
}

func (s *capturedFinalScreen) Bounds() image.Rectangle { return s.bounds }
func (s *capturedFinalScreen) DrawImage(source *ebiten.Image, options *ebiten.DrawImageOptions) {
	s.source = source
	s.options = *options
}

func TestFinalScreenUsesOriginalSurfaceAtDeviceResolution(t *testing.T) {
	layout := render.TouchLayout(390, 700)
	g := &Game{renderer: &render.Renderer{Layout: layout}}
	g.Layout(390, 700)
	surface := g.ensureSurface()
	defer surface.Deallocate()
	intermediate := ebiten.NewImage(390, 700)
	defer intermediate.Deallocate()
	for _, dpr := range []float64{1, 2, 3} {
		final := &capturedFinalScreen{bounds: image.Rect(0, 0, int(390*dpr), int(700*dpr))}
		g.DrawFinalScreen(final, intermediate, ebiten.GeoM{})
		if final.source != surface {
			t.Fatal("final pass sampled the reduced window image instead of the original")
		}
		if final.options.Filter != ebiten.FilterNearest {
			t.Fatal("final pass smooths pixel art")
		}
		x, y := final.options.GeoM.Apply(float64(layout.ScreenW), float64(layout.ScreenH))
		if math.Abs(x-float64(final.bounds.Dx())) > 0.001 || math.Abs(y-float64(final.bounds.Dy())) > 0.001 {
			t.Fatalf("DPR %v: surface does not fill device screen: %v, %v", dpr, x, y)
		}
		// Device pixels / DPR must still map to the same logical input point.
		x, y = final.options.GeoM.Apply(240, float64(layout.ControlsTop()+180))
		lx, ly := g.toLogical(int(math.Round(x/dpr)), int(math.Round(y/dpr)))
		if math.Abs(float64(lx-240)) > 1 || math.Abs(float64(ly-layout.ControlsTop()-180)) > 1 {
			t.Fatalf("DPR %v: pointer mapping changed: %d, %d", dpr, lx, ly)
		}
	}
}
