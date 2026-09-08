package render

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestRadiantItemsOverrideBaseSprites(t *testing.T) {
	s, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	base := &Sprites{frames: map[string][]*ebiten.Image{}, flipped: map[string][]*ebiten.Image{}}
	if err := base.loadDir("custom"); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"food", "elixir", "scroll", "sword", "portal"} {
		want := image.Pt(32, 32)
		if role == "portal" {
			want = image.Pt(56, 64)
		}
		got := s.Frame(role, 0)
		if got == nil || len(s.frames[role]) != 1 || got.Bounds().Size() != want {
			t.Fatalf("%s: missing override or unwanted tile scaling", role)
		}
		// Verify that applying the theme really replaces each legacy frame.
		// GPU pixel reads require a running game; terrainpreview checks those.
		previous := base.Frame(role, 0)
		if err := base.loadDir("custom/radiant"); err != nil {
			t.Fatal(err)
		}
		if base.Frame(role, 0) == previous {
			t.Fatalf("%s: theme did not replace base", role)
		}
		if err := base.loadDir("custom"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPortalAnchorsArchitectureAndProtectsNeighbouringItems(t *testing.T) {
	for _, p := range []domain.Point{{}, {X: 21, Y: 8}, {X: -2, Y: 3}} {
		b := portalBounds(p)
		if b.Size() != image.Pt(56, 64) || b.Min.X+b.Max.X != (2*p.X+1)*TileSize || b.Max.Y != (p.Y+1)*TileSize {
			t.Fatalf("exit %v: off-centre artwork %v", p, b)
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				neighbour := image.Rect((p.X+dx)*TileSize, (p.Y+dy)*TileSize, (p.X+dx+1)*TileSize, (p.Y+dy+1)*TileSize)
				centre := neighbour.Min.Add(image.Pt(TileSize/2, TileSize/2))
				if gateAlpha(centre, []image.Rectangle{neighbour}) > .13 {
					t.Fatalf("gate does not protect neighbour %d,%d", dx, dy)
				}
			}
		}
	}
}
