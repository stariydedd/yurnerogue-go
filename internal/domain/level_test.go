package domain

import "testing"

// Уровень генерируется случайно, поэтому инварианты проверяются на выборке.
const samples = 30

func TestLevelHasMinimumRooms(t *testing.T) {
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		count := 0
		for _, r := range l.Rooms {
			if r != nil {
				count++
			}
		}
		if count < MinRoomsCount {
			t.Fatalf("комнат %d, ожидалось не меньше %d", count, MinRoomsCount)
		}
	}
}

func TestLevelRoomsWithinBounds(t *testing.T) {
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		for _, r := range l.Rooms {
			if r == nil {
				continue
			}
			// Стены лежат кольцом снаружи пола, они тоже должны попадать на карту.
			if r.X-1 < 0 || r.Y-1 < 0 || r.X+r.W >= Cols || r.Y+r.H >= Rows {
				t.Fatalf("комната вылезла за карту: %+v", *r)
			}
			if r.W < MinW || r.W > MaxW || r.H < MinH || r.H > MaxH {
				t.Fatalf("размер комнаты вне допустимого: %+v", *r)
			}
		}
	}
}

func TestLevelIsConnected(t *testing.T) {
	// generateValid обязан выдать остовное дерево: коридоров на один меньше,
	// чем комнат. Проверяем, что из стартовой комнаты достижимы все остальные.
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		start := l.Rooms[l.StartRoomIdx]
		reached := reachableCells(l, Point{start.X, start.Y})
		for idx, r := range l.Rooms {
			if r == nil || idx == l.StartRoomIdx {
				continue
			}
			if !reached[Point{r.X, r.Y}] {
				t.Fatalf("комната %d недостижима из стартовой", idx)
			}
		}
	}
}

// reachableCells — обход в ширину по проходимым клеткам уровня.
func reachableCells(l *Level, from Point) map[Point]bool {
	walkable := func(x, y int) bool {
		return InBounds(x, y) &&
			(IsAnyRoomFloorCell(x, y, l.Rooms) || InPassageCenter(x, y, l.Passages))
	}
	seen := map[Point]bool{from: true}
	queue := []Point{from}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range dirs4 {
			n := Point{c.X + d.X, c.Y + d.Y}
			if !seen[n] && walkable(n.X, n.Y) {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return seen
}

func TestDoorsLieOnRoomWalls(t *testing.T) {
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		if len(l.Doors) == 0 {
			t.Fatal("на уровне нет ни одной двери")
		}
		for d := range l.Doors {
			if !IsAnyRoomWallCell(d.X, d.Y, l.Rooms) {
				t.Fatalf("дверь %+v не на стене комнаты", d)
			}
			if !InPassageCenter(d.X, d.Y, l.Passages) {
				t.Fatalf("дверь %+v не на линии коридора", d)
			}
		}
	}
}

func TestExitNotAdjacentToDoors(t *testing.T) {
	// Портал не должен стоять вплотную к входу, иначе игрок проваливается
	// на следующий уровень, едва войдя в комнату.
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		room := l.RoomAt(l.Exit.X, l.Exit.Y)
		if room == nil {
			t.Fatalf("выход %+v вне пола комнат", l.Exit)
		}
		validExists := false
		for y := room.Y; y < room.Y+room.H; y++ {
			for x := room.X; x < room.X+room.W; x++ {
				if l.farFromDoors(Point{x, y}) {
					validExists = true
				}
			}
		}
		if validExists && !l.farFromDoors(l.Exit) {
			t.Fatalf("выход %+v встал у двери, хотя есть подходящие клетки", l.Exit)
		}
	}
}

func TestExitNotOnStartRoomOrEnemy(t *testing.T) {
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		if l.RoomAt(l.Exit.X, l.Exit.Y) == l.Rooms[l.StartRoomIdx] {
			t.Fatal("выход оказался в стартовой комнате")
		}
		for _, op := range l.AllOpponents() {
			if op.X == l.Exit.X && op.Y == l.Exit.Y {
				t.Fatalf("враг стоит на клетке выхода %+v", l.Exit)
			}
		}
	}
}

func TestNoEnemiesInStartRoom(t *testing.T) {
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		if n := len(l.Rooms[l.StartRoomIdx].Enemies); n != 0 {
			t.Fatalf("в стартовой комнате %d врагов", n)
		}
	}
}

func TestItemsAvoidExitCell(t *testing.T) {
	for i := 0; i < samples; i++ {
		l := NewLevel(1)
		player := NewPerson()
		l.GenerateItems(player)
		for _, r := range l.Rooms {
			if r == nil {
				continue
			}
			for _, it := range r.Items {
				if it.X == l.Exit.X && it.Y == l.Exit.Y {
					t.Fatalf("предмет лёг на клетку выхода %+v", l.Exit)
				}
			}
		}
	}
}

func TestSeedsDifferBetweenLevels(t *testing.T) {
	// В WASM-сборке Python интерпретатор стартовал с одинаковым состоянием
	// генератора; в Go рантайм засеивает его сам, уровни обязаны различаться.
	seeds := map[int64]bool{}
	for i := 0; i < 5; i++ {
		seeds[NewLevel(1).Seed] = true
	}
	if len(seeds) < 2 {
		t.Fatal("сиды уровней не различаются")
	}
}
