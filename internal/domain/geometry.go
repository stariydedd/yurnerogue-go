package domain

// Геометрия карты: единственное место, где определены комнаты, коридоры и двери.
//
// Коридор хранится как прямоугольник с запасом в одну клетку по периметру;
// проходимая «центральная линия» — внутренность этого прямоугольника.
// Дверь — клетка центральной линии коридора, лежащая на кольце стен комнаты.

// Point — клетка карты.
type Point struct {
	X, Y int
}

// Rect — прямоугольник коридора: левый верхний угол плюс размеры.
type Rect struct {
	X, Y, W, H int
}

// Room — комната: пол начинается в (X, Y), стены лежат кольцом снаружи.
type Room struct {
	X, Y, W, H int

	Enemies []*Opponent
}

// PassageCenterCells возвращает клетки центральной линии коридора.
func (r Rect) PassageCenterCells() []Point {
	cells := make([]Point, 0, (r.W-2)*(r.H-2))
	for y := r.Y + 1; y < r.Y+r.H-1; y++ {
		for x := r.X + 1; x < r.X+r.W-1; x++ {
			cells = append(cells, Point{x, y})
		}
	}
	return cells
}

// ContainsCenter сообщает, лежит ли клетка на центральной линии коридора.
func (r Rect) ContainsCenter(x, y int) bool {
	return x >= r.X+1 && x < r.X+r.W-1 && y >= r.Y+1 && y < r.Y+r.H-1
}

// InPassageCenter — лежит ли клетка на центральной линии хотя бы одного коридора.
func InPassageCenter(x, y int, passages []Rect) bool {
	for _, p := range passages {
		if p.ContainsCenter(x, y) {
			return true
		}
	}
	return false
}

// IsFloorCell — входит ли клетка в пол комнаты.
func (r *Room) IsFloorCell(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// IsWallCell — является ли клетка внешней стеной комнаты.
func (r *Room) IsWallCell(x, y int) bool {
	if x == r.X-1 || x == r.X+r.W {
		return y >= r.Y && y < r.Y+r.H
	}
	if y == r.Y-1 || y == r.Y+r.H {
		return x >= r.X && x < r.X+r.W
	}
	return false
}

// IsAnyRoomFloorCell — входит ли клетка в пол хотя бы одной комнаты.
func IsAnyRoomFloorCell(x, y int, rooms []*Room) bool {
	for _, r := range rooms {
		if r != nil && r.IsFloorCell(x, y) {
			return true
		}
	}
	return false
}

// IsAnyRoomWallCell — является ли клетка стеной хотя бы одной комнаты.
func IsAnyRoomWallCell(x, y int, rooms []*Room) bool {
	for _, r := range rooms {
		if r != nil && r.IsWallCell(x, y) {
			return true
		}
	}
	return false
}

// IsCorridorCell — клетка тропы: центральная линия коридора вне пола комнат
// (двери считаются тропой).
func IsCorridorCell(x, y int, rooms []*Room, passages []Rect) bool {
	return InPassageCenter(x, y, passages) && !IsAnyRoomFloorCell(x, y, rooms)
}

// DoorCells — все двери уровня: клетки центральных линий коридоров на стенах комнат.
func DoorCells(rooms []*Room, passages []Rect) map[Point]bool {
	doors := make(map[Point]bool)
	for _, p := range passages {
		for _, c := range p.PassageCenterCells() {
			if IsAnyRoomWallCell(c.X, c.Y, rooms) {
				doors[c] = true
			}
		}
	}
	return doors
}

// PathCells — клетки троп для отрисовки: коридоры (и двери) за вычетом пола комнат.
func PathCells(rooms []*Room, passages []Rect) map[Point]bool {
	cells := make(map[Point]bool)
	for _, p := range passages {
		for _, c := range p.PassageCenterCells() {
			if !IsAnyRoomFloorCell(c.X, c.Y, rooms) {
				cells[c] = true
			}
		}
	}
	return cells
}

// InBounds — лежит ли клетка в пределах карты.
func InBounds(x, y int) bool {
	return x >= 0 && x < Cols && y >= 0 && y < Rows
}
