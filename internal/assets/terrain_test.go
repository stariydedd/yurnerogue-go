package assets

import (
	"image/png"
	"testing"
)

func TestRadiantTerrainSheets(t *testing.T) {
	for _, role := range []string{"floor", "decor", "tree", "bush", "ruins", "verge", "moss", "meadow", "trail"} {
		t.Run(role, func(t *testing.T) {
			f, err := FS.Open("custom/radiant/" + role + ".4.png")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			img, err := png.Decode(f)
			if err != nil {
				t.Fatal(err)
			}
			w, h := 32, 32
			switch role {
			case "tree":
				w, h = 96, 128
			case "meadow", "trail":
				w, h = 128, 128
			case "bush":
				w, h = 48, 40
			case "ruins":
				w, h = 64, 80
			case "verge":
				w, h = 64, 32
			case "moss":
				w, h = 128, 64
			}
			if img.Bounds().Dx() != w*4 || img.Bounds().Dy() != h {
				t.Fatalf("must contain four %dx%d frames, got %v", w, h, img.Bounds())
			}
			plant := role == "decor" || role == "tree" || role == "bush" || role == "ruins" || role == "verge" || role == "moss"
			for frame := 0; frame < 4; frame++ {
				transparent, solid := 0, 0
				for y := 0; y < h; y++ {
					for x := 0; x < w; x++ {
						_, _, _, a := img.At(frame*w+x, y).RGBA()
						if a == 0 {
							transparent++
						}
						// Generated cutouts retain slightly translucent edge/interior
						// pixels; terrain itself must remain fully opaque.
						if a == 65535 || (plant && a >= 240*257) {
							solid++
						}
						if plant && (x == 0 || y == 0 || x == w-1 || y == h-1) && a != 0 {
							t.Fatalf("frame %d: plant touches cell boundary at %d,%d", frame, x, y)
						}
					}
				}
				if plant && (transparent < 100 || solid < 30) {
					t.Fatalf("frame %d: missing plant silhouette or real alpha", frame)
				}
				if !plant && solid != w*h {
					t.Fatalf("frame %d: ground must cover the entire cell", frame)
				}
			}
		})
	}
}
