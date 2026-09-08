package render

import (
	"image"
	"image/color"
	"testing"
)

func TestPickupOutlineFollowsSilhouetteAndPreservesInterior(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 17, 27))
	inside := color.NRGBA{R: 30, G: 160, B: 90, A: 240}
	src.SetNRGBA(13, 23, inside)
	// Isolated faint glow must not acquire its own outline.
	src.SetNRGBA(10, 20, color.NRGBA{R: 80, G: 150, A: 30})
	dst := outlineSprite(src, src.Bounds())
	if dst.Bounds() != image.Rect(0, 0, 9, 9) {
		t.Fatal(dst.Bounds())
	}
	if got := dst.NRGBAAt(4, 4); got != inside {
		t.Fatalf("changed interior: %v", got)
	}
	for y := 3; y <= 5; y++ {
		for x := 3; x <= 5; x++ {
			if x == 4 && y == 4 {
				continue
			}
			if got := dst.NRGBAAt(x, y); got != (color.NRGBA{A: 255}) {
				t.Fatalf("missing black edge %d,%d: %v", x, y, got)
			}
		}
	}
	if dst.NRGBAAt(0, 0).A != 0 || dst.NRGBAAt(8, 8).A != 0 || dst.NRGBAAt(2, 1).A != 0 {
		t.Fatal("rectangular background or outlined faint glow")
	}
}

func TestPickupOutlineDoesNotClipFrameEdgesOrSampleAdjacentFrames(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(2, 1, color.NRGBA{G: 255, A: 255})
	dst := outlineSprite(src, image.Rect(0, 0, 2, 2))
	if dst.NRGBAAt(0, 0) != (color.NRGBA{A: 255}) {
		t.Fatal("clipped top-left outline")
	}
	if dst.NRGBAAt(3, 2).A != 0 {
		t.Fatal("sampled neighbouring sprite frame")
	}
}

func TestGroundPickupFramesAreSeparateFromUIAndReloadWithTheme(t *testing.T) {
	s, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"food", "elixir", "scroll", "sword"} {
		frames := s.groundPickups[role]
		if len(frames) != 1 || frames[0].Bounds().Size() != image.Pt(34, 34) {
			t.Fatalf("%s: wrong ground frame", role)
		}
		if s.Frame(role, 0).Bounds().Size() != image.Pt(32, 32) {
			t.Fatalf("%s: changed UI frame", role)
		}
		before := frames[0]
		if err := s.loadDir("custom/radiant"); err != nil {
			t.Fatal(err)
		}
		if before == s.groundPickups[role][0] {
			t.Fatalf("%s: stale outline", role)
		}
	}
	for _, role := range []string{"player", "portal", "floor", "heart"} {
		if len(s.groundPickups[role]) != 0 {
			t.Fatalf("outlined non-pickup %s", role)
		}
	}
}
