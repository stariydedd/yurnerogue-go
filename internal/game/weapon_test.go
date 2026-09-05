package game

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestReplaceWeaponPreservesOldWeapon(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pos     domain.Point
		blocked bool
		drop    bool
	}{
		{name: "corridor", pos: domain.Point{X: 10, Y: 6}, drop: true},
		{name: "doorway", pos: domain.Point{X: 8, Y: 6}, drop: true},
		{name: "blocked corridor", pos: domain.Point{X: 10, Y: 6}, blocked: true},
		{name: "full room", pos: domain.Point{X: 6, Y: 6}, blocked: true},
		{name: "room", pos: domain.Point{X: 6, Y: 6}, drop: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			room := &domain.Room{X: 4, Y: 4, W: 4, H: 4}
			level := &domain.Level{
				Rooms:    []*domain.Room{room},
				Passages: []domain.Rect{{X: 7, Y: 5, W: 6, H: 3}},
				Exit:     domain.Point{X: 4, Y: 4},
			}
			level.Doors = domain.DoorCells(level.Rooms, level.Passages)
			p := domain.NewPerson()
			p.X, p.Y = tc.pos.X, tc.pos.Y
			old := &domain.Item{Type: domain.ItemWeapon, Name: "Old blade", StrengthEffect: 30}
			p.Weapon = old
			// Даже полный рюкзак получает свободный слот при экипировке.
			for i := 0; i < domain.MaxBackpackItemsPerType; i++ {
				p.PickUpItem(domain.NewWeapon())
			}
			replacement := p.Backpack[0]
			others := append([]*domain.Item(nil), p.Backpack[1:]...)
			if tc.blocked {
				for y := p.Y - 1; y <= p.Y+1; y++ {
					for x := p.X - 1; x <= p.X+1; x++ {
						if (domain.Point{X: x, Y: y}) != tc.pos {
							if domain.IsAnyRoomFloorCell(x, y, level.Rooms) || domain.InPassageCenter(x, y, level.Passages) {
								level.Items = append(level.Items, &domain.Item{Type: domain.ItemFood, X: x, Y: y})
							}
						}
					}
				}
			}
			s := &domain.Session{Player: p, Level: level, LevelNum: 1}
			g := &Game{session: s, state: StatePlaying}
			g.HandleKey(ebiten.KeyH)
			g.HandleKey(ebiten.Key1)

			if p.Weapon != replacement || g.state != StatePlaying {
				t.Fatal("selected weapon must be equipped and menu closed")
			}
			backpackCount, floorCount := 0, 0
			for _, item := range p.Backpack {
				if item == replacement {
					t.Fatal("equipped weapon remains in backpack")
				}
				if item == old {
					backpackCount++
				}
			}
			for i, item := range others {
				if p.Backpack[i] != item {
					t.Fatal("other backpack weapons changed")
				}
			}
			for _, item := range level.Items {
				if item == old {
					floorCount++
				}
			}
			if tc.drop {
				if floorCount != 1 || backpackCount != 0 || (domain.Point{X: old.X, Y: old.Y}) == tc.pos {
					t.Fatal("old weapon must be dropped once beside the player")
				}
				if !strings.Contains(s.Message, "Dropped Old blade.") {
					t.Fatalf("incorrect drop message: %q", s.Message)
				}
			} else {
				if backpackCount != 1 || floorCount != 0 || len(p.Backpack) != domain.MaxBackpackItemsPerType {
					t.Fatal("old weapon must occupy the freed backpack slot exactly once")
				}
				if !strings.Contains(s.Message, "Stowed Old blade in backpack.") || strings.Contains(s.Message, "Dropped") {
					t.Fatalf("incorrect fallback message: %q", s.Message)
				}
			}
		})
	}
}
