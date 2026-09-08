package main

import (
	"image"
	"image/color"
	"testing"
)

func TestFitSpritePreservesAspectAlphaAndCentres(t *testing.T) {
	src := image.NewNRGBA(image.Rect(20, 30, 120, 130))
	for y := 50; y < 90; y++ {
		for x := 40; x < 100; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 80, G: 190, B: 30, A: 240})
		}
	}
	dst, err := fitSprite(src, 32, 32, 2)
	if err != nil {
		t.Fatal(err)
	}
	if dst.Bounds() != image.Rect(0, 0, 32, 32) {
		t.Fatal(dst.Bounds())
	}
	var bounds image.Rectangle
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if dst.NRGBAAt(x, y).A > 0 {
				if x == 0 || y == 0 || x == 31 || y == 31 {
					t.Fatal("lost transparent padding")
				}
				bounds = bounds.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if bounds.Dx() <= bounds.Dy() || bounds.Min.X+bounds.Max.X < 31 || bounds.Min.X+bounds.Max.X > 33 || bounds.Min.Y+bounds.Max.Y < 31 || bounds.Min.Y+bounds.Max.Y > 33 {
		t.Fatalf("stretched or off-centre: %v", bounds)
	}
	c := dst.NRGBAAt(16, 16)
	if c.A < 235 || c.A > 245 || c.G < 185 || c.G > 195 {
		t.Fatalf("changed interior: %v", c)
	}
}

func TestFitSpriteRejectsMissingAlphaOrSilhouette(t *testing.T) {
	for _, src := range []image.Image{image.NewUniform(color.White), image.NewNRGBA(image.Rect(0, 0, 8, 8))} {
		// Restrict Uniform's otherwise enormous bounds.
		bounded := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		for y := 0; y < 8; y++ {
			for x := 0; x < 8; x++ {
				bounded.Set(x, y, src.At(x, y))
			}
		}
		if _, err := fitSprite(bounded, 32, 32, 2); err == nil {
			t.Fatal("accepted invalid cutout")
		}
	}
}
