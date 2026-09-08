package render

import (
	"image"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestAllHeroAnimationFramesHaveSeparateOutlines(t *testing.T) {
	s, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	roles := []string{"player"}
	for _, kind := range domain.AllOpponentTypes {
		roles = append(roles, kind.SpriteRole())
	}
	for _, role := range roles {
		base := s.frames[role]
		if len(base) == 0 || len(s.worldHeroes[role]) != len(base) {
			t.Fatalf("%s: lost animation frames", role)
		}
		for i, original := range base {
			got, outlined := s.heroFrame(role, i)
			if !outlined || got == original || got.Bounds().Size() != original.Bounds().Size().Add(image.Pt(2, 2)) {
				t.Fatalf("%s frame %d: missing padding or changed UI frame", role, i)
			}
			for _, tick := range []int{i + len(base), i - len(base)} {
				wrapped, _ := s.heroFrame(role, tick)
				if wrapped != got {
					t.Fatalf("%s: frame wrapping changed", role)
				}
			}
		}
	}
	previous, _ := s.heroFrame("player", 0)
	if err := s.loadDir("custom"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.heroFrame("player", 0); got == previous {
		t.Fatal("stale hero outline after reload")
	}
	for _, role := range []string{"portal", "food", "floor", "heart", "missing"} {
		if _, outlined := s.heroFrame(role, 0); outlined {
			t.Fatalf("outlined non-hero %s", role)
		}
	}
}

func TestHeroOutlineKeepsEveryPixelAnchoredWhenMirrored(t *testing.T) {
	for _, size := range []image.Point{{X: 37, Y: 40}, {X: 50, Y: 36}, {X: 31, Y: 33}} {
		for _, facing := range []int{-1, 1} {
			base := heroDrawOptions(size, 16, 10, 171, 29, facing, false)
			outlined := heroDrawOptions(size.Add(image.Pt(2, 2)), 16, 10, 171, 29, facing, true)
			for y := 0; y <= size.Y; y++ {
				for x := 0; x <= size.X; x++ {
					bx, by := base.GeoM.Apply(float64(x), float64(y))
					ox, oy := outlined.GeoM.Apply(float64(x+1), float64(y+1))
					if bx != ox || by != oy {
						t.Fatalf("size %v facing %d: shifted %d,%d", size, facing, x, y)
					}
				}
			}
			if outlined.ColorScale.A() != 1 || outlined.ColorScale.R() != float32(worldSpriteBrightness) {
				t.Fatal("wrong contrast or changed opacity")
			}
		}
	}
}
