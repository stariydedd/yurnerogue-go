package render

import (
	"image"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestGateAlphaSoftLocalCutaway(t *testing.T) {
	b := image.Rect(10, 20, 40, 50)
	if gateAlpha(image.Pt(20, 30), nil) != 1 {
		t.Fatal("unoccupied gate faded")
	}
	protected := []image.Rectangle{b}
	if gateAlpha(image.Pt(20, 30), protected) > .13 {
		t.Fatal("item silhouette not protected")
	}
	previous := float32(0)
	for x := 40; x <= 46; x++ {
		a := gateAlpha(image.Pt(x, 30), protected)
		if a < previous || a < .12 || a > 1 {
			t.Fatal("invalid fade", a)
		}
		previous = a
		if a != gateAlpha(image.Pt(x, 30), []image.Rectangle{b, b}) {
			t.Fatal("duplicate objects compound fade")
		}
	}
	if previous != 1 {
		t.Fatal("distant stone not opaque")
	}
}

func TestGateUnrevealedExitDoesNotDraw(t *testing.T) {
	s := domain.NewSessionSeed(21)
	// No sprites or destination: a hidden exit must return before using either.
	(&Renderer{}).drawGate(nil, s, domain.Visibility{}, 0, 0, 0)
}

func TestGateForegroundTracksVisibleObjectsWithoutTerrainInvalidation(t *testing.T) {
	sprites, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	r := &Renderer{sprites: sprites}
	s := domain.NewSessionSeed(21)
	s.Level.Rooms = []*domain.Room{{X: 8, Y: 6, W: 16, H: 9}}
	s.Player.X, s.Player.Y = 10, 10
	s.Level.Exit = domain.Point{X: 18, Y: 11}
	s.Level.Items = nil
	b := portalBounds(s.Level.Exit)
	p := domain.Point{X: 18, Y: 10}
	vis := domain.Visibility{Visible: map[domain.Point]bool{p: true}}
	count := func() int { return len(r.gateForeground(s, vis, 0, b)) }
	if count() != 0 {
		t.Fatal("empty gate protected")
	}
	s.Level.Items = []*domain.Item{{Type: domain.ItemElixir, X: p.X, Y: p.Y}}
	if count() != 1 {
		t.Fatal("visible loot not protected")
	}
	delete(vis.Visible, p)
	if count() != 0 {
		t.Fatal("hidden loot leaks into cutaway")
	}
	vis.Visible[p] = true
	s.Level.Items = nil
	if count() != 0 {
		t.Fatal("removed loot leaves stale cutaway")
	}
	op := domain.NewOpponent(domain.Zombie)
	op.X, op.Y, op.IsVisible = p.X, p.Y, true
	s.Level.Rooms[0].Enemies = []*domain.Opponent{op}
	if count() != 1 {
		t.Fatal("visible enemy not protected")
	}
	op.IsVisible = false
	if count() != 0 {
		t.Fatal("invisible enemy leaks into cutaway")
	}
	op.IsVisible = true
	delete(vis.Visible, p)
	if count() != 0 {
		t.Fatal("fogged enemy leaks into cutaway")
	}
	s.Level.Rooms[0].Enemies = nil
	s.Player.X, s.Player.Y = p.X, p.Y
	if count() != 1 {
		t.Fatal("player not protected")
	}
	s.Player.X, s.Player.Y = 10, 10
	if count() != 0 {
		t.Fatal("moved player leaves stale cutaway")
	}
}
