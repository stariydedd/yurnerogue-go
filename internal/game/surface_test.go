package game

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

type capturedFinalScreen struct {
	ebiten.FinalScreen
	bounds  image.Rectangle
	source  *ebiten.Image
	options ebiten.DrawImageOptions
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
