package domain

// FogRadius — дальность обзора игрока в коридоре.
const FogRadius = 8

// Visibility — что игрок видит сейчас и что помнит.
type Visibility struct {
	// Visible — клетки, видимые целиком (пол, предметы, враги).
	Visible map[Point]bool
	// Explored — клетки уже посещённых комнат: помнятся только стены.
	Explored map[Point]bool
}

// bresenhamLine — клетки прямой между двумя точками.
func bresenhamLine(x0, y0, x1, y1 int) []Point {
	var cells []Point
	dx, dy := abs(x1-x0), abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		cells = append(cells, Point{x0, y0})
		if x0 == x1 && y0 == y1 {
			return cells
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// fieldOfView — клетки в радиусе, до которых доходит луч: стены обзор
// останавливают, пустота обрывает луч.
func fieldOfView(px, py int, grid Grid, radius int) map[Point]bool {
	visible := map[Point]bool{}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > radius*radius {
				continue
			}
			tx, ty := px+dx, py+dy
			if !InBounds(tx, ty) {
				continue
			}
			for _, c := range bresenhamLine(px, py, tx, ty) {
				if !InBounds(c.X, c.Y) || grid.At(c.X, c.Y) == SymEmpty {
					break
				}
				visible[c] = true
				if grid.At(c.X, c.Y) == SymWall {
					break
				}
			}
		}
	}
	return visible
}

// ComputeVisibility обновляет список посещённых комнат и возвращает видимость.
//
// В комнате игрок видит её целиком вместе со стенами; в коридоре — только свой
// коридор в пределах радиуса и двери рядом. Посещённые комнаты запоминаются
// стенами, поэтому карта постепенно раскрывается.
func (s *Session) ComputeVisibility(grid Grid) Visibility {
	rooms := s.Level.Rooms
	px, py := s.Player.X, s.Player.Y

	roomIdx := -1
	for i, r := range rooms {
		if r != nil && r.IsFloorCell(px, py) {
			roomIdx = i
			break
		}
	}

	// Стоя в дверном проёме, игрок считается вошедшим в комнату.
	onDoor := false
	if roomIdx < 0 {
		for i, r := range rooms {
			if r != nil && r.IsWallCell(px, py) {
				roomIdx, onDoor = i, true
				break
			}
		}
	}
	if roomIdx >= 0 {
		s.VisitedRooms[roomIdx] = true
	}

	vis := Visibility{Visible: map[Point]bool{}, Explored: map[Point]bool{}}

	if roomIdx >= 0 {
		r := rooms[roomIdx]
		for y := r.Y - 1; y <= r.Y+r.H; y++ {
			for x := r.X - 1; x <= r.X+r.W; x++ {
				if InBounds(x, y) {
					vis.Visible[Point{x, y}] = true
				}
			}
		}
		if onDoor {
			// Из проёма виден ещё и коридор, в который он ведёт.
			corridor := s.playerCorridorCells(px, py)
			for c := range fieldOfView(px, py, grid, FogRadius) {
				if corridor[c] {
					vis.Visible[c] = true
				}
			}
		}
	} else {
		corridor := s.playerCorridorCells(px, py)
		for c := range fieldOfView(px, py, grid, FogRadius) {
			switch {
			case grid.At(c.X, c.Y) == SymDoor:
				// Дверь видна, если примыкает к коридору игрока.
				for _, d := range dirs4 {
					if corridor[Point{c.X + d.X, c.Y + d.Y}] {
						vis.Visible[c] = true
						break
					}
				}
			case corridor[c]:
				vis.Visible[c] = true
			}
		}
	}

	for idx := range s.VisitedRooms {
		if idx == roomIdx || rooms[idx] == nil {
			continue
		}
		r := rooms[idx]
		for y := r.Y - 1; y <= r.Y+r.H; y++ {
			for x := r.X - 1; x <= r.X+r.W; x++ {
				p := Point{x, y}
				if InBounds(x, y) && !vis.Visible[p] {
					vis.Explored[p] = true
				}
			}
		}
	}
	return vis
}

// playerCorridorCells — все клетки коридоров, в которых сейчас стоит игрок
// (или к которым примыкает, стоя в проёме).
func (s *Session) playerCorridorCells(px, py int) map[Point]bool {
	cells := map[Point]bool{}
	for _, p := range s.Level.Passages {
		inside := p.ContainsCenter(px, py)
		if !inside {
			// Проём: коридор считается «своим», если начинается рядом.
			for _, d := range dirs4 {
				if p.ContainsCenter(px+d.X, py+d.Y) {
					inside = true
					break
				}
			}
		}
		if inside {
			for _, c := range p.PassageCenterCells() {
				cells[c] = true
			}
		}
	}
	return cells
}
