package main

import (
	"image"
	"image/color"
	"testing"
)

func TestPackFrameOrderAndAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 138, 148))
	colors := []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 128}, {}}
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			src.SetNRGBA(x+10, y+20, colors[(y/64)*2+x/64])
		}
	}
	dst, err := pack(src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if dst.Bounds() != image.Rect(0, 0, 128, 32) {
		t.Fatal(dst.Bounds())
	}
	for x := 0; x < 128; x++ {
		for y := 0; y < 32; y++ {
			if got := dst.NRGBAAt(x, y); got != colors[x/32] {
				t.Fatalf("pixel %d,%d: %v, want %v", x, y, got, colors[x/32])
			}
		}
	}
}

func TestPackRejectsMalformedSheets(t *testing.T) {
	for _, size := range []image.Point{{63, 63}, {65, 65}, {128, 64}} {
		if _, err := pack(image.NewNRGBA(image.Rectangle{Max: size}), 0); err == nil {
			t.Fatalf("accepted %v", size)
		}
	}
}

func TestPackPlantPadding(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			src.SetNRGBA(x, y, color.NRGBA{G: 100, A: 255})
		}
	}
	dst, err := pack(src, 1)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 128; x++ {
			want := uint8(255)
			if y == 0 || y == 31 || x%32 == 0 || x%32 == 31 {
				want = 0
			}
			if got := dst.NRGBAAt(x, y).A; got != want {
				t.Fatalf("alpha at %d,%d: %d, want %d", x, y, got, want)
			}
		}
	}
	for _, padding := range []int{-1, 16} {
		if _, err := pack(src, padding); err == nil {
			t.Fatalf("accepted padding %d", padding)
		}
	}
}
