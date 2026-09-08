// terrainpack mechanically packs generated 2x2 sheets into game sprite strips.
// Artwork and alpha are preserved; no drawing,
// background removal, palette changes or runtime generation happens here.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func pack(src image.Image, padding int) (*image.NRGBA, error) {
	if src.Bounds().Dx() != src.Bounds().Dy() {
		return nil, fmt.Errorf("expected square sheet")
	}
	return packFrames(src, 32, 32, padding, false)
}

func packFrames(src image.Image, width, height, padding int, trim bool) (*image.NRGBA, error) {
	if padding < 0 || padding >= 16 {
		return nil, fmt.Errorf("padding must be between 0 and 15px")
	}
	b := src.Bounds()
	if b.Dx()%2 != 0 || b.Dy()%2 != 0 || b.Dx() < 64 || b.Dy() < 64 {
		return nil, fmt.Errorf("expected an even 2x2 sheet at least 64px, got %v", b)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width*4, height))
	sw, sh := b.Dx()/2, b.Dy()/2
	for frame := 0; frame < 4; frame++ {
		x, y := b.Min.X+(frame%2)*sw, b.Min.Y+(frame/2)*sh
		sr := image.Rect(x, y, x+sw, y+sh)
		dr := image.Rect(frame*width+padding, padding, (frame+1)*width-padding, height-padding)
		if trim {
			// Crop transparent margins and fit without stretching. Keep the
			// original alpha; the threshold only ignores faint extraction dust
			// when measuring the silhouette's bounding box.
			bounds := image.Rectangle{}
			for sy := sr.Min.Y; sy < sr.Max.Y; sy++ {
				for sx := sr.Min.X; sx < sr.Max.X; sx++ {
					_, _, _, a := src.At(sx, sy).RGBA()
					if a >= 32*257 {
						bounds = bounds.Union(image.Rect(sx, sy, sx+1, sy+1))
					}
				}
			}
			if bounds.Empty() {
				return nil, fmt.Errorf("frame %d has no silhouette", frame)
			}
			sr = bounds.Inset(-2).Intersect(sr)
			w, h := dr.Dx(), dr.Dy()
			if sr.Dx()*h > sr.Dy()*w {
				h = sr.Dy() * w / sr.Dx()
			} else {
				w = sr.Dx() * h / sr.Dy()
			}
			dr = image.Rect(dr.Min.X+(dr.Dx()-w)/2, dr.Max.Y-h, dr.Min.X+(dr.Dx()+w)/2, dr.Max.Y)
		}
		draw.CatmullRom.Scale(dst, dr, src, sr, draw.Src, nil)
	}
	return dst, nil
}

func run() error {
	out := "internal/assets/custom/radiant"
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	for _, spec := range []struct {
		role, source string
		w, h         int
		trim         bool
	}{
		{"floor", "floor", 32, 32, false}, {"decor", "decor", 32, 32, false},
		{"tree", "tree", 96, 128, true},
		{"bush", "decor", 48, 40, true}, {"ruins", "ruins", 64, 80, true},
		{"verge", "verge", 64, 32, true},
		{"moss", "moss", 128, 64, true},
		{"meadow", "meadow", 128, 128, false}, {"trail", "trail", 128, 128, false},
	} {
		padding := 0
		if spec.role == "decor" || spec.trim {
			padding = 2
		}
		if err := convert(filepath.Join("docs/art/radiant/sources", spec.source+".png"),
			filepath.Join(out, spec.role+".4.png"), spec.w, spec.h, padding, spec.trim); err != nil {
			return fmt.Errorf("%s: %w", spec.role, err)
		}
	}
	return nil
}

func convert(input, output string, width, height, padding int, trim bool) error {
	f, err := os.Open(input)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	dst, err := packFrames(src, width, height, padding, trim)
	if err != nil {
		return err
	}
	f, err = os.Create(output)
	if err != nil {
		return err
	}
	if err := png.Encode(f, dst); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
