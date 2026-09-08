package assets

import (
	"image/color"
	"image/png"
	"testing"
)

func TestRadiantItemCutouts(t *testing.T) {
	for _, role := range []string{"food", "elixir", "scroll", "sword", "portal"} {
		t.Run(role, func(t *testing.T) {
			f, err := FS.Open("custom/radiant/" + role + ".png")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			img, err := png.Decode(f)
			if err != nil {
				t.Fatal(err)
			}
			w, h := 32, 32
			if role == "portal" {
				w, h = 56, 64
			}
			if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
				t.Fatalf("wrong dimensions: %v", img.Bounds())
			}
			transparent, solid, brightBlue := 0, 0, 0
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					_, _, _, a := img.At(x, y).RGBA()
					c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
					if c.A >= 230 && c.B >= 190 && c.G >= 175 && int(c.B)-int(c.R) >= 35 {
						brightBlue++
					}
					if a == 0 {
						transparent++
					}
					if a >= 230*257 {
						solid++
					}
					if (x == 0 || y == 0 || x == w-1 || y == h-1) && a != 0 {
						t.Fatalf("nontransparent border at %d,%d", x, y)
					}
				}
			}
			if transparent < w*h/5 || solid < 20 {
				t.Fatalf("missing alpha or opaque silhouette: transparent=%d solid=%d", transparent, solid)
			}
			if role == "portal" {
				// Broad luminous blue detail must survive packing to game size.
				if brightBlue < 100 {
					t.Fatalf("portal glow lost at runtime scale: %d bright blue pixels", brightBlue)
				}
				// The opening is above the solid, grounded landing platform.
				_, _, _, a := img.At(w/2, h*3/8).RGBA()
				if a != 0 {
					t.Fatal("Portal opening must be genuinely transparent")
				}
				_, _, _, platform := img.At(w/2, h*3/4).RGBA()
				if platform < 230*257 {
					t.Fatal("Portal landing platform must remain solid at its anchor")
				}
			}
		})
	}
}
