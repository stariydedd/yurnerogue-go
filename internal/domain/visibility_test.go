package domain

import "testing"

func TestVisibilityShowsWholeRoomFromInside(t *testing.T) {
	s := cleanSession(t)
	room := s.Level.RoomAt(s.Player.X, s.Player.Y)
	vis := s.ComputeVisibility(s.BuildGrid(true))

	for y := room.Y; y < room.Y+room.H; y++ {
		for x := room.X; x < room.X+room.W; x++ {
			if !vis.Visible[Point{x, y}] {
				t.Fatalf("клетка комнаты (%d,%d) не видна изнутри", x, y)
			}
		}
	}
	// Стены комнаты тоже видны, иначе она рисуется без границ.
	if !vis.Visible[Point{room.X - 1, room.Y}] {
		t.Fatal("стена комнаты должна быть видна")
	}
}

func TestVisibilityHidesUnvisitedRooms(t *testing.T) {
	s := cleanSession(t)
	current := s.Level.RoomAt(s.Player.X, s.Player.Y)
	vis := s.ComputeVisibility(s.BuildGrid(true))

	for _, r := range s.Level.Rooms {
		if r == nil || r == current {
			continue
		}
		center := Point{r.X + r.W/2, r.Y + r.H/2}
		if vis.Visible[center] || vis.Explored[center] {
			t.Fatalf("непосещённая комната видна в точке %+v", center)
		}
	}
}

func TestVisitedRoomIsRememberedAsExplored(t *testing.T) {
	s := cleanSession(t)
	first := s.Level.RoomAt(s.Player.X, s.Player.Y)
	s.ComputeVisibility(s.BuildGrid(true))

	// Переносим игрока в другую комнату и пересчитываем видимость.
	var other *Room
	for _, r := range s.Level.Rooms {
		if r != nil && r != first {
			other = r
			break
		}
	}
	if other == nil {
		t.Skip("на карте одна комната")
	}
	s.Player.X, s.Player.Y = other.X, other.Y
	vis := s.ComputeVisibility(s.BuildGrid(true))

	corner := Point{first.X - 1, first.Y - 1}
	if !vis.Explored[corner] {
		t.Fatal("покинутая комната должна помниться как исследованная")
	}
	if vis.Visible[Point{first.X, first.Y}] {
		t.Fatal("пол покинутой комнаты не должен быть виден")
	}
}

func TestVisibilityInCorridorIsLimited(t *testing.T) {
	// В коридоре игрок видит свой коридор, но не внутренности чужих комнат.
	s, door := findDoorApproach(t, func(s *Session, d Point) bool {
		return IsCorridorCell(d.X-1, d.Y, s.Level.Rooms, s.Level.Passages) &&
			IsCorridorCell(d.X-2, d.Y, s.Level.Rooms, s.Level.Passages)
	})
	s.Player.X, s.Player.Y = door.X-2, door.Y
	vis := s.ComputeVisibility(s.BuildGrid(true))

	if !vis.Visible[Point{door.X - 1, door.Y}] {
		t.Fatal("соседняя клетка своего коридора должна быть видна")
	}
	for _, r := range s.Level.Rooms {
		if r == nil {
			continue
		}
		center := Point{r.X + r.W/2, r.Y + r.H/2}
		if vis.Visible[center] && !r.IsFloorCell(s.Player.X, s.Player.Y) {
			t.Fatalf("из коридора видна внутренность комнаты %+v", center)
		}
	}
}
