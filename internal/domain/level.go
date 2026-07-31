package domain

import "math/rand"

// Level — уровень подземелья: комнаты, коридоры, точка старта и выход.
type Level struct {
	Num      int
	Seed     int64
	Rooms    []*Room // длина GridDim*GridDim, nil — пустая ячейка сетки
	Passages []Rect
	Doors    map[Point]bool // считаются один раз, единое определение из geometry

	StartRoomIdx int
	Exit         Point

	roomsCount int
}

// NewLevel генерирует уровень с номером num: комнаты, связную сеть коридоров,
// врагов и выход.
func NewLevel(num int) *Level {
	l := &Level{Num: num}
	l.generateValid()
	l.Doors = DoorCells(l.Rooms, l.Passages)
	l.pickStartRoom()
	l.generateOpponents()
	l.Exit = l.exitPosition()
	return l
}

// generateValid перегенерирует уровень, пока комнаты не окажутся связными:
// коридоров должно быть ровно на одну меньше, чем комнат (остовное дерево).
func (l *Level) generateValid() {
	for {
		l.generateRooms()
		if l.roomsCount < MinRoomsCount {
			continue
		}
		passages, connections := l.generatePassages()
		if connections == l.roomsCount-1 {
			l.Passages = passages
			return
		}
	}
}

// generateRooms раскладывает комнаты по сетке GridDim x GridDim; часть ячеек
// остаётся пустой с вероятностью 1-ProbRoom.
func (l *Level) generateRooms() {
	// Seed записывается для лидерборда и отладки. Пересеивать генератор, как
	// это делала Python-версия, не нужно: рантайм Go засеивает его случайно
	// при старте — в том числе в WASM, где CPython давал один и тот же уровень.
	l.Seed = int64(rand.Intn(10_000_000_000-100) + 100)

	rooms := make([]*Room, 0, GridDim*GridDim)
	for r := 0; r < GridDim; r++ {
		for c := 0; c < GridDim; c++ {
			if rand.Float64() > ProbRoom {
				rooms = append(rooms, nil)
				continue
			}
			availW, availH := CellW-2*Pad, CellH-2*Pad
			if availW < MinW || availH < MinH {
				rooms = append(rooms, nil)
				continue
			}
			w := MinW + rand.Intn(min(MaxW, availW)-MinW+1)
			h := MinH + rand.Intn(min(MaxH, availH)-MinH+1)
			x := c*CellW + Pad + rand.Intn(availW-w+1)
			y := r*CellH + Pad + rand.Intn(availH-h+1)
			x = clamp(x, 0, Cols-w)
			y = clamp(y, 0, Rows-h)
			rooms = append(rooms, &Room{X: x, Y: y, W: w, H: h})
		}
	}

	l.Rooms = rooms
	l.roomsCount = 0
	for _, r := range rooms {
		if r != nil {
			l.roomsCount++
		}
	}
}

// roomEdges — пары индексов соседних по сетке комнат, которые можно соединить.
func (l *Level) roomEdges() [][2]int {
	var edges [][2]int
	for i := 0; i < GridDim; i++ {
		for j := 0; j < GridDim-1; j++ {
			a, b := i*GridDim+j, i*GridDim+j+1
			if l.Rooms[a] != nil && l.Rooms[b] != nil {
				edges = append(edges, [2]int{a, b})
			}
		}
	}
	for i := 0; i < GridDim-1; i++ {
		for j := 0; j < GridDim; j++ {
			a, b := i*GridDim+j, i*GridDim+j+GridDim
			if l.Rooms[a] != nil && l.Rooms[b] != nil {
				edges = append(edges, [2]int{a, b})
			}
		}
	}
	return edges
}

// dsu — система непересекающихся множеств для алгоритма Крускала.
type dsu struct {
	parent, rank []int
}

func newDSU(n int) *dsu {
	d := &dsu{parent: make([]int, n), rank: make([]int, n)}
	for i := range d.parent {
		d.parent[i] = i
	}
	return d
}

func (d *dsu) find(v int) int {
	if d.parent[v] != v {
		d.parent[v] = d.find(d.parent[v])
	}
	return d.parent[v]
}

func (d *dsu) connected(a, b int) bool { return d.find(a) == d.find(b) }

func (d *dsu) union(a, b int) {
	ra, rb := d.find(a), d.find(b)
	if ra == rb {
		return
	}
	if d.rank[rb] >= d.rank[ra] {
		d.parent[ra] = rb
	} else {
		d.parent[rb] = ra
	}
	if d.rank[ra] == d.rank[rb] {
		d.rank[rb]++
	}
}

// generatePassages строит коридоры, связывающие все комнаты в одну сеть.
func (l *Level) generatePassages() ([]Rect, int) {
	edges := l.roomEdges()
	rand.Shuffle(len(edges), func(i, j int) { edges[i], edges[j] = edges[j], edges[i] })

	var passages []Rect
	connections := 0
	d := newDSU(len(l.Rooms))
	for _, e := range edges {
		if d.connected(e[0], e[1]) {
			continue
		}
		d.union(e[0], e[1])
		connections++
		if abs(e[0]-e[1]) == 1 {
			passages = l.horizontalPassage(e, passages)
		} else {
			passages = l.verticalPassage(e, passages)
		}
	}
	return passages, connections
}

// addSegment добавляет сегмент коридора, расширенный на клетку по краям:
// проходимой остаётся центральная линия (см. geometry).
func addSegment(passages []Rect, x, y, w, h int) []Rect {
	return append(passages, Rect{X: x - 1, Y: y - 1, W: w + 2, H: h + 2})
}

// turnBetween выбирает координату поворота Г-образного коридора строго между
// a и b. Если промежутка нет, возвращает середину: rand.Intn(0) паникует, а
// паника в WASM убивает всю игру.
func turnBetween(a, b int) int {
	lo, hi := min(a, b), max(a, b)
	if hi-lo < 2 {
		return (lo + hi) / 2
	}
	return lo + 1 + rand.Intn(hi-lo-1)
}

// horizontalPassage соединяет две горизонтально соседние комнаты: прямой
// коридор, если выходы совпали по высоте, иначе Г-образный из трёх сегментов.
func (l *Level) horizontalPassage(e [2]int, passages []Rect) []Rect {
	first, second := l.Rooms[e[0]], l.Rooms[e[1]]

	firstX := first.X + first.W
	firstY := first.Y + rand.Intn(first.H)
	secondX := second.X - 1
	secondY := second.Y + rand.Intn(second.H)

	if firstY == secondY {
		return addSegment(passages, firstX, firstY, abs(secondX-firstX)+1, 1)
	}
	turn := turnBetween(firstX, secondX)
	passages = addSegment(passages, firstX, firstY, abs(turn-firstX)+1, 1)
	passages = addSegment(passages, turn, min(firstY, secondY), 1, abs(secondY-firstY)+1)
	passages = addSegment(passages, turn, secondY, abs(secondX-turn)+1, 1)
	return passages
}

// verticalPassage соединяет две вертикально соседние комнаты.
func (l *Level) verticalPassage(e [2]int, passages []Rect) []Rect {
	first, second := l.Rooms[e[0]], l.Rooms[e[1]]

	firstY := first.Y + first.H
	firstX := first.X + rand.Intn(first.W)
	secondY := second.Y - 1
	secondX := second.X + rand.Intn(second.W)

	if firstX == secondX {
		return addSegment(passages, firstX, firstY, 1, abs(secondY-firstY)+1)
	}
	turn := turnBetween(firstY, secondY)
	passages = addSegment(passages, firstX, firstY, 1, abs(turn-firstY)+1)
	passages = addSegment(passages, min(firstX, secondX), turn, abs(secondX-firstX)+1, 1)
	passages = addSegment(passages, secondX, turn, 1, abs(secondY-turn)+1)
	return passages
}

// pickStartRoom выбирает стартовую комнату игрока.
func (l *Level) pickStartRoom() {
	var valid []int
	for i, r := range l.Rooms {
		if r != nil {
			valid = append(valid, i)
		}
	}
	l.StartRoomIdx = pick(valid)
}

// PlayerStart — случайная клетка стартовой комнаты.
func (l *Level) PlayerStart() Point {
	return l.Rooms[l.StartRoomIdx].RandomCell()
}

// RandomCell — случайная клетка пола комнаты.
func (r *Room) RandomCell() Point {
	return Point{r.X + rand.Intn(r.W), r.Y + rand.Intn(r.H)}
}

// generateOpponents расселяет врагов по всем комнатам, кроме стартовой.
func (l *Level) generateOpponents() {
	maxPerRoom := MaxMonstersPerRoom + l.Num/LevelUpdateDifficulty
	for i, room := range l.Rooms {
		if room == nil || i == l.StartRoomIdx {
			continue
		}
		count := rand.Intn(maxPerRoom + 1)
		for n := 0; n < count; n++ {
			// До 16 попыток найти клетку, не занятую другим врагом.
			for attempt := 0; attempt < 16; attempt++ {
				c := room.RandomCell()
				if room.enemyAt(c) != nil {
					continue
				}
				op := RandomOpponent(l.Num)
				op.X, op.Y = c.X, c.Y
				room.Enemies = append(room.Enemies, op)
				break
			}
		}
	}
}

// enemyAt — живой враг в клетке комнаты или nil.
func (r *Room) enemyAt(c Point) *Opponent {
	for _, op := range r.Enemies {
		if op.IsAlive() && op.X == c.X && op.Y == c.Y {
			return op
		}
	}
	return nil
}

// exitPosition выбирает клетку выхода: случайная комната (не стартовая) и
// клетка не ближе чем через одну от любой двери и не занятая врагом. Клетки
// перечисляются явно, поэтому подходящая находится всегда, когда существует.
func (l *Level) exitPosition() Point {
	var candidates []*Room
	for i, r := range l.Rooms {
		if r != nil && i != l.StartRoomIdx {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) == 0 {
		for _, r := range l.Rooms {
			if r != nil {
				candidates = append(candidates, r)
			}
		}
	}
	room := pick(candidates)

	var free, valid []Point
	for y := room.Y; y < room.Y+room.H; y++ {
		for x := room.X; x < room.X+room.W; x++ {
			c := Point{x, y}
			if room.enemyAt(c) != nil {
				continue
			}
			free = append(free, c)
			if l.farFromDoors(c) {
				valid = append(valid, c)
			}
		}
	}
	if len(valid) > 0 {
		return pick(valid)
	}
	if len(free) > 0 {
		return pick(free)
	}
	return room.RandomCell()
}

// farFromDoors — клетка дальше чем на одну (по Чебышёву) от каждой двери:
// портал у самого входа проваливал бы игрока, едва он вошёл.
func (l *Level) farFromDoors(c Point) bool {
	for d := range l.Doors {
		if max(abs(c.X-d.X), abs(c.Y-d.Y)) <= 1 {
			return false
		}
	}
	return true
}

// GenerateItems наполняет комнаты предметами, кроме стартовой; клетку выхода
// обходит — предмет под порталом не виден и подбирается без сообщения.
func (l *Level) GenerateItems(player *Person) {
	maxItems := MaxConsumablesPerRoom - l.Num/LevelUpdateDifficulty
	if maxItems < 1 {
		maxItems = 1
	}
	for i, room := range l.Rooms {
		if room == nil || i == l.StartRoomIdx {
			continue
		}
		count := rand.Intn(maxItems + 1)
		for n := 0; n < count; n++ {
			for attempt := 0; attempt < 16; attempt++ {
				c := room.RandomCell()
				if c == l.Exit || room.itemAt(c) != nil {
					continue
				}
				item := RandomItem(player)
				item.X, item.Y = c.X, c.Y
				room.Items = append(room.Items, item)
				break
			}
		}
	}
}

// itemAt — предмет в клетке комнаты или nil.
func (r *Room) itemAt(c Point) *Item {
	for _, it := range r.Items {
		if it.X == c.X && it.Y == c.Y {
			return it
		}
	}
	return nil
}

// AllOpponents — все враги уровня.
func (l *Level) AllOpponents() []*Opponent {
	var out []*Opponent
	for _, r := range l.Rooms {
		if r != nil {
			out = append(out, r.Enemies...)
		}
	}
	return out
}

// RoomAt — комната, содержащая клетку, или nil.
func (l *Level) RoomAt(x, y int) *Room {
	for _, r := range l.Rooms {
		if r != nil && r.IsFloorCell(x, y) {
			return r
		}
	}
	return nil
}
