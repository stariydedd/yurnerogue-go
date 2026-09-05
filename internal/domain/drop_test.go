package domain

import "testing"

func TestDropInPassagesAndPickUp(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start Point
		cells []Point
		room  *Room
	}{
		{name: "horizontal", start: Point{10, 10}, cells: []Point{{9, 10}, {10, 10}, {11, 10}}},
		{name: "vertical", start: Point{10, 10}, cells: []Point{{10, 9}, {10, 10}, {10, 11}}},
		{name: "turn", start: Point{10, 10}, cells: []Point{{10, 9}, {10, 10}, {11, 10}}},
		{name: "onto doorway", start: Point{9, 10}, cells: []Point{{9, 10}, {10, 10}}, room: &Room{X: 11, Y: 9, W: 3, H: 3}},
		{name: "map edge", start: Point{0, 0}, cells: []Point{{0, 0}, {1, 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Session{Player: NewPerson(), Level: &Level{Exit: Point{99, 44}}, VisitedRooms: map[int]bool{}}
			s.Player.X, s.Player.Y = tc.start.X, tc.start.Y
			if tc.room != nil {
				s.Level.Rooms = []*Room{tc.room}
			}
			for _, c := range tc.cells {
				s.Level.Passages = append(s.Level.Passages, Rect{X: c.X - 1, Y: c.Y - 1, W: 3, H: 3})
			}
			s.Level.Doors = DoorCells(s.Level.Rooms, s.Level.Passages)
			weapon := NewWeapon()
			if !s.DropItemNearPlayer(weapon) {
				t.Fatal("drop failed with a free adjacent passage cell")
			}
			dx, dy := weapon.X-tc.start.X, weapon.Y-tc.start.Y
			if abs(dx)+abs(dy) != 1 || !s.CanMoveTo(weapon.X, weapon.Y) {
				t.Fatalf("weapon is not one step away: (%d,%d)", weapon.X, weapon.Y)
			}
			grid := s.BuildGrid(false)
			if grid.At(weapon.X, weapon.Y) != SymItem || !s.ComputeVisibility(grid).Visible[Point{weapon.X, weapon.Y}] {
				t.Fatal("dropped weapon must be represented on the map and visible")
			}
			for i := 0; i < MaxBackpackItemsPerType; i++ {
				s.Player.PickUpItem(NewWeapon())
			}
			if dx != 0 {
				s.MoveX(dx)
			} else {
				s.MoveY(dy)
			}
			s.CheckItemPickup()
			if len(s.Level.Items) != 1 || s.Level.Items[0] != weapon {
				t.Fatal("full backpack must leave the weapon on the floor")
			}
			s.Player.EquipWeapon(s.Player.Backpack[0]) // Освобождаем слот.
			s.CheckItemPickup()
			s.CheckItemPickup() // Повторная проверка не должна дублировать предмет.
			if len(s.Level.Items) != 0 || len(s.Player.Backpack) != MaxBackpackItemsPerType ||
				s.Player.Backpack[MaxBackpackItemsPerType-1] != weapon {
				t.Fatal("weapon must move from floor to backpack exactly once")
			}
			s.Player.X, s.Player.Y = tc.start.X, tc.start.Y
			want := SymCorridor
			if s.Level.Doors[Point{weapon.X, weapon.Y}] {
				want = SymDoor
			}
			if got := s.BuildGrid(false).At(weapon.X, weapon.Y); got != want {
				t.Fatalf("pickup changed passage terrain: got %c, want %c", got, want)
			}
		})
	}
}

func TestDropAvoidsOccupiedCellsAndPortal(t *testing.T) {
	s := &Session{Player: NewPerson(), Level: &Level{Exit: Point{10, 9}}}
	s.Player.X, s.Player.Y = 10, 10
	room := &Room{X: 9, Y: 9, W: 3, H: 3}
	s.Level.Rooms = []*Room{room}
	enemy := NewOpponent(Zombie)
	enemy.X, enemy.Y = 9, 10
	room.Enemies = []*Opponent{enemy}
	for y := 9; y <= 11; y++ {
		for x := 9; x <= 11; x++ {
			c := Point{x, y}
			if c != s.Level.Exit && c != (Point{10, 10}) && c != (Point{9, 10}) && c != (Point{11, 10}) {
				s.Level.Items = append(s.Level.Items, &Item{Type: ItemFood, X: x, Y: y})
			}
		}
	}
	weapon := NewWeapon()
	if !s.DropItemNearPlayer(weapon) || weapon.X != 11 || weapon.Y != 10 {
		t.Fatal("drop must use the only unoccupied non-portal cell")
	}
	before := len(s.Level.Items)
	if s.DropItemNearPlayer(NewWeapon()) || len(s.Level.Items) != before {
		t.Fatal("drop must fail without changing floor items when all neighbors are blocked")
	}
}

func TestDropDoesNotCrossWallCorner(t *testing.T) {
	s := &Session{Player: NewPerson(), Level: &Level{
		Exit:     Point{99, 44},
		Passages: []Rect{{X: 9, Y: 9, W: 3, H: 3}, {X: 10, Y: 10, W: 3, H: 3}},
	}}
	s.Player.X, s.Player.Y = 10, 10
	if s.DropItemNearPlayer(NewWeapon()) || len(s.Level.Items) != 0 {
		t.Fatal("a diagonally adjacent passage behind walls must not receive the weapon")
	}
}

func TestDropUsesFreeDiagonalInRoom(t *testing.T) {
	s := &Session{Player: NewPerson(), Level: &Level{
		Rooms: []*Room{{X: 9, Y: 9, W: 3, H: 3}}, Exit: Point{99, 44},
	}}
	s.Player.X, s.Player.Y = 10, 10
	for _, d := range dirs4 {
		s.Level.Items = append(s.Level.Items, &Item{Type: ItemFood, X: 10 + d.X, Y: 10 + d.Y})
	}
	weapon := NewWeapon()
	if !s.DropItemNearPlayer(weapon) || abs(weapon.X-10) != 1 || abs(weapon.Y-10) != 1 {
		t.Fatal("free diagonal must remain usable when cardinal neighbors contain items")
	}
}

func TestDescendingLeavesFloorItemsBehind(t *testing.T) {
	s := NewSession()
	weapon := NewWeapon()
	s.Level.Items = append(s.Level.Items, weapon)
	s.Player.X, s.Player.Y = s.Level.Exit.X, s.Level.Exit.Y
	if !s.CheckExit() {
		t.Fatal("player must descend from the portal")
	}
	for _, item := range s.Level.Items {
		if item == weapon {
			t.Fatal("floor item from the previous level followed the player")
		}
	}
}
