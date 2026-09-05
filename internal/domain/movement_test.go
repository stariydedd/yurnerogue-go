package domain

import "testing"

// cleanSession — сессия без врагов и предметов: бег становится детерминированным.
func cleanSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	s.Level.Items = nil
	for _, r := range s.Level.Rooms {
		if r != nil {
			r.Enemies = nil
		}
	}
	return s
}

// findDoorApproach ищет дверь, к которой слева ведёт прямой коридор, и
// возвращает её координаты. Раскладка карты случайна, поэтому подходящий
// участок находится не на каждом уровне — пробуем несколько сессий.
func findDoorApproach(t *testing.T, want func(s *Session, d Point) bool) (*Session, Point) {
	t.Helper()
	for attempt := 0; attempt < 60; attempt++ {
		s := cleanSession(t)
		for d := range s.Level.Doors {
			if want(s, d) {
				return s, d
			}
		}
	}
	t.Skip("на сгенерированных картах не нашлось подходящего участка")
	return nil, Point{}
}

func TestRunStopsAtWall(t *testing.T) {
	s := cleanSession(t)
	room := s.Level.Rooms[s.Level.StartRoomIdx]
	// Ставим игрока к левому краю комнаты: бег вправо детерминирован.
	s.Player.X, s.Player.Y = room.X, room.Y
	startX := s.Player.X

	s.Run(1, 0)

	if s.Player.X <= startX {
		t.Fatalf("бег не сдвинул игрока: %d -> %d", startX, s.Player.X)
	}
	// Упёрся: справа стена, портал, дверь либо проём сбоку.
	nx, ny := s.Player.X+1, s.Player.Y
	blocked := !s.CanMoveTo(nx, ny) ||
		(nx == s.Level.Exit.X && ny == s.Level.Exit.Y) ||
		s.Level.Doors[Point{nx, ny}] ||
		(IsAnyRoomFloorCell(s.Player.X, s.Player.Y, s.Level.Rooms) && s.doorBeside(1, 0))
	if !blocked {
		t.Fatalf("бег встал на клетке (%d,%d), хотя впереди свободно", s.Player.X, s.Player.Y)
	}
}

func TestRunFromCorridorEndsInDoorway(t *testing.T) {
	// Бег по коридору заканчивается в дверном проёме: дверь и есть конец коридора.
	s, door := findDoorApproach(t, func(s *Session, d Point) bool {
		return IsCorridorCell(d.X-1, d.Y, s.Level.Rooms, s.Level.Passages) &&
			IsCorridorCell(d.X-2, d.Y, s.Level.Rooms, s.Level.Passages) &&
			!s.Level.Doors[Point{d.X - 1, d.Y - 1}] && !s.Level.Doors[Point{d.X - 1, d.Y + 1}] &&
			s.Level.Exit != Point{d.X - 1, d.Y}
	})

	s.Player.X, s.Player.Y = door.X-2, door.Y
	s.Run(1, 0)

	if (Point{s.Player.X, s.Player.Y}) != door {
		t.Fatalf("бег закончился в (%d,%d), ожидался проём %+v", s.Player.X, s.Player.Y, door)
	}
}

func TestRunInRoomStopsBeforeDoor(t *testing.T) {
	// Бег внутри комнаты в сторону двери встаёт на клетке перед ней.
	s, door := findDoorApproach(t, func(s *Session, d Point) bool {
		return IsAnyRoomFloorCell(d.X+1, d.Y, s.Level.Rooms) &&
			IsAnyRoomFloorCell(d.X+2, d.Y, s.Level.Rooms) &&
			!s.Level.Doors[Point{d.X + 1, d.Y - 1}] && !s.Level.Doors[Point{d.X + 1, d.Y + 1}] &&
			s.Level.Exit != Point{d.X + 1, d.Y} && s.Level.Exit != Point{d.X + 2, d.Y}
	})

	s.Player.X, s.Player.Y = door.X+2, door.Y
	s.Run(-1, 0)

	if s.Player.X != door.X+1 || s.Player.Y != door.Y {
		t.Fatalf("бег встал в (%d,%d), ожидалась клетка перед дверью (%d,%d)",
			s.Player.X, s.Player.Y, door.X+1, door.Y)
	}
}

func TestRunNextToDoorPassesThrough(t *testing.T) {
	// Стоя вплотную к двери, шаг в её сторону проносит через дверь в коридор.
	s, door := findDoorApproach(t, func(s *Session, d Point) bool {
		return IsAnyRoomFloorCell(d.X+1, d.Y, s.Level.Rooms) &&
			IsCorridorCell(d.X-1, d.Y, s.Level.Rooms, s.Level.Passages) &&
			IsCorridorCell(d.X-2, d.Y, s.Level.Rooms, s.Level.Passages)
	})

	s.Player.X, s.Player.Y = door.X+1, door.Y
	s.Run(-1, 0)

	if s.Player.X > door.X-1 {
		t.Fatalf("бег застрял в (%d,%d), дверь %+v не пройдена", s.Player.X, s.Player.Y, door)
	}
}

func TestRunIntoWallDoesNothing(t *testing.T) {
	// Бег в непроходимую сторону — «невозможный ход», нулевое движение.
	s := cleanSession(t)
	room := s.Level.Rooms[s.Level.StartRoomIdx]
	s.Player.X, s.Player.Y = room.X, room.Y
	// Слева от левого края комнаты стена (если там не оказалось двери).
	if s.Level.Doors[Point{room.X - 1, room.Y}] {
		t.Skip("в стартовой комнате дверь ровно слева от угла")
	}
	start := Point{s.Player.X, s.Player.Y}

	s.Run(-1, 0)

	if (Point{s.Player.X, s.Player.Y}) != start {
		t.Fatalf("игрок сдвинулся в стену: %+v -> (%d,%d)", start, s.Player.X, s.Player.Y)
	}
	if s.Message != "Can't move that way." {
		t.Fatalf("сообщение %q, ожидалось «Can't move that way.»", s.Message)
	}
}

func TestRunIntoAdjacentEnemyReports(t *testing.T) {
	// Бег в упор к врагу не начинается, но и не выглядит мёртвым нажатием.
	s := cleanSession(t)
	room := s.Level.Rooms[s.Level.StartRoomIdx]
	s.Player.X, s.Player.Y = room.X, room.Y

	op := NewOpponent(Zombie)
	op.X, op.Y = s.Player.X+1, s.Player.Y
	room.Enemies = append(room.Enemies, op)
	start := Point{s.Player.X, s.Player.Y}

	s.Run(1, 0)

	if (Point{s.Player.X, s.Player.Y}) != start {
		t.Fatalf("игрок сдвинулся, хотя впереди враг")
	}
	if s.Message != "An enemy is in the way." {
		t.Fatalf("сообщение %q, ожидалось «An enemy is in the way.»", s.Message)
	}
}

func TestMoveAttacksEnemyInTargetCell(t *testing.T) {
	s := cleanSession(t)
	room := s.Level.Rooms[s.Level.StartRoomIdx]
	s.Player.X, s.Player.Y = room.X, room.Y

	op := NewOpponent(Zombie)
	op.X, op.Y = s.Player.X+1, s.Player.Y
	room.Enemies = append(room.Enemies, op)
	start := Point{s.Player.X, s.Player.Y}

	if !s.MoveX(1) {
		t.Fatal("ход должен состояться: в клетке враг, значит атака")
	}
	if (Point{s.Player.X, s.Player.Y}) != start {
		t.Fatal("игрок не должен вставать на клетку врага")
	}
	if s.Stats.AttacksMade != 1 {
		t.Fatalf("атак засчитано %d, ожидалась 1", s.Stats.AttacksMade)
	}
}

func TestMoveUpdatesFacingEvenIntoWall(t *testing.T) {
	s := cleanSession(t)
	room := s.Level.Rooms[s.Level.StartRoomIdx]
	s.Player.X, s.Player.Y = room.X, room.Y
	s.Player.Facing = 1

	s.MoveX(-1) // шаг в стену
	if s.Player.Facing != -1 {
		t.Fatal("разворот должен происходить даже при шаге в стену")
	}
	s.MoveX(1)
	if s.Player.Facing != 1 {
		t.Fatal("разворот вправо не сработал")
	}
}

func TestPickupPutsItemInBackpack(t *testing.T) {
	s := cleanSession(t)
	item := NewFood(s.Player)
	item.X, item.Y = s.Player.X, s.Player.Y
	s.Level.Items = append(s.Level.Items, item)

	s.CheckItemPickup()

	if len(s.Player.Backpack) != 1 {
		t.Fatalf("в рюкзаке %d предметов, ожидался 1", len(s.Player.Backpack))
	}
	if len(s.Level.Items) != 0 {
		t.Fatal("предмет должен исчезнуть с пола")
	}
}

func TestDropItemNearPlayerUsesAdjacentCell(t *testing.T) {
	s := cleanSession(t)
	weapon := NewWeapon()
	playerPos := Point{s.Player.X, s.Player.Y}

	s.DropItemNearPlayer(weapon)

	if len(s.Level.Items) != 1 || s.Level.Items[0] != weapon {
		t.Fatal("dropped weapon must be placed on the room floor")
	}
	dx := abs(weapon.X - playerPos.X)
	dy := abs(weapon.Y - playerPos.Y)
	if dx == 0 && dy == 0 {
		t.Fatal("dropped weapon must not be placed under the player")
	}
	if max(dx, dy) != 1 {
		t.Fatalf("weapon dropped at (%d,%d), expected a cell adjacent to (%d,%d)",
			weapon.X, weapon.Y, playerPos.X, playerPos.Y)
	}
}

func TestCheckExitDescendsToNextLevel(t *testing.T) {
	s := cleanSession(t)
	s.Player.X, s.Player.Y = s.Level.Exit.X, s.Level.Exit.Y

	if !s.CheckExit() {
		t.Fatal("стоя на портале, игрок должен спуститься")
	}
	if s.LevelNum != 2 {
		t.Fatalf("уровень %d, ожидался 2", s.LevelNum)
	}
	if !IsAnyRoomFloorCell(s.Player.X, s.Player.Y, s.Level.Rooms) {
		t.Fatal("после спуска игрок должен стоять на полу новой комнаты")
	}
}
