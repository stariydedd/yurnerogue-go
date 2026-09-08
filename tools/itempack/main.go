// itempack mechanically trims and scales generated transparent item cutouts.
// It preserves colours/alpha; it does not draw or remove backgrounds.
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

type spriteSpec struct {
	role                   string
	width, height, padding int
}

var specs = []spriteSpec{
	{"food", 32, 32, 2},
	{"elixir", 32, 32, 3},
	{"scroll", 32, 32, 2},
	{"sword", 32, 32, 1},
	{"portal", 56, 64, 2},
}

func fitSprite(src image.Image, width, height, padding int) (*image.NRGBA, error) {
	if width <= 0 || height <= 0 || padding < 1 || 2*padding >= min(width, height) {
		return nil, fmt.Errorf("invalid sprite size or padding")
	}
	var silhouette image.Rectangle
	transparent := 0
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a == 0 {
				transparent++
			}
			// Ignore faint outer glow when measuring the sprite, but do not
			// change alpha values within the retained source rectangle.
			if a >= 128*257 {
				silhouette = silhouette.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if transparent == 0 || silhouette.Empty() {
		return nil, fmt.Errorf("source must have real transparency and a solid silhouette")
	}
	sr := silhouette.Inset(-2).Intersect(src.Bounds())
	w, h := width-2*padding, height-2*padding
	if sr.Dx()*h > sr.Dy()*w {
		h = max(1, sr.Dy()*w/sr.Dx())
	} else {
		w = max(1, sr.Dx()*h/sr.Dy())
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	dr := image.Rect((width-w)/2, (height-h)/2, (width-w)/2+w, (height-h)/2+h)
	draw.CatmullRom.Scale(dst, dr, src, sr, draw.Src, nil)
	return dst, nil
}

func convert(spec spriteSpec) error {
	source := spec.role + ".png"
	f, err := os.Open(filepath.Join("docs/art/radiant/items/sources", source))
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	dst, err := fitSprite(src, spec.width, spec.height, spec.padding)
	if err != nil {
		return err
	}
	out := filepath.Join("internal/assets/custom/radiant", spec.role+".png")
	f, err = os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(f, dst); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Printf("%s: %dx%d\n", out, spec.width, spec.height)
	return nil
}

func main() {
	for _, spec := range specs {
		if err := convert(spec); err != nil {
			fmt.Fprintln(os.Stderr, spec.role+":", err)
			os.Exit(1)
		}
	}
}
