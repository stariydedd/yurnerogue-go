package render

import (
	"image"
	"testing"
)

func TestHUDPortraitUsesStableIntegerPlacementForEveryFrame(t *testing.T) {
	sprites, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		box   image.Rectangle
		scale int
	}{
		{"desktop", image.Rect(12, 726, 100, 826), 2},
		{"mobile", image.Rect(10, 637, 66, 709), 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &Renderer{sprites: sprites}
			var anchor image.Point
			for tick := 0; tick < 8*AnimFrameTicks*2; tick++ {
				frame := sprites.Frame("player", r.AnimTick())
				scale, position := portraitPlacement(frame.Bounds().Size(), test.box)
				if scale != test.scale {
					t.Fatalf("tick %d: scale %d, want %d", tick, scale, test.scale)
				}
				if tick == 0 {
					anchor = position
				}
				if position != anchor {
					t.Fatalf("portrait anchor moved at tick %d", tick)
				}
				painted := image.Rectangle{Min: position, Max: position.Add(frame.Bounds().Size().Mul(scale))}
				if !painted.In(test.box.Inset(6)) {
					t.Fatalf("frame touches border: %v", painted)
				}
				if frame != sprites.Frame("player", (tick/AnimFrameTicks)%8) {
					t.Fatal("portrait animation timing changed")
				}
				r.Tick()
			}
		})
	}
}

func TestHUDPortraitRejectsInvalidOrTooSmallSlots(t *testing.T) {
	for _, frame := range []image.Point{{0, 40}, {37, 0}, {37, 40}} {
		if scale, _ := portraitPlacement(frame, image.Rect(0, 0, 20, 20)); scale != 0 {
			t.Fatal("portrait exceeds tiny slot")
		}
	}
}
