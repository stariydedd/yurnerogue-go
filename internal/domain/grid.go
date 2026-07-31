package domain

// Grid — карта уровня в символах: строки по Y, столбцы по X.
type Grid [][]byte

// At возвращает символ клетки; за пределами карты — SymEmpty.
func (g Grid) At(x, y int) byte {
	if !InBounds(x, y) {
		return SymEmpty
	}
	return g[y][x]
}

// set пишет символ, молча игнорируя выход за карту.
func (g Grid) set(x, y int, sym byte) {
	if InBounds(x, y) {
		g[y][x] = sym
	}
}

// BuildGrid собирает карту в символах по состоянию уровня: стены и пол комнат,
// коридоры, двери, предметы, врагов, выход и игрока.
//
// Нужна для отрисовки и расчёта видимости. Проверять проходимость через неё
// не стоит — для этого есть CanMoveTo, который смотрит геометрию напрямую.
func (s *Session) BuildGrid(withOpponents bool) Grid {
	grid := make(Grid, Rows)
	for y := range grid {
		grid[y] = make([]byte, Cols)
		for x := range grid[y] {
			grid[y][x] = SymEmpty
		}
	}

	rooms, passages := s.Level.Rooms, s.Level.Passages

	for _, r := range rooms {
		if r == nil {
			continue
		}
		left, right := r.X-1, r.X+r.W
		top, bottom := r.Y-1, r.Y+r.H
		for x := left; x <= right; x++ {
			grid.set(x, top, SymWall)
			grid.set(x, bottom, SymWall)
		}
		for y := top; y <= bottom; y++ {
			grid.set(left, y, SymWall)
			grid.set(right, y, SymWall)
		}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				grid.set(x, y, SymRoomFloor)
			}
		}
	}

	// Коридоры поверх стен, затем двери там, где коридор пересекает стену.
	var corridor []Point
	for _, p := range passages {
		corridor = append(corridor, p.PassageCenterCells()...)
	}
	for _, c := range corridor {
		if !IsAnyRoomFloorCell(c.X, c.Y, rooms) {
			grid.set(c.X, c.Y, SymCorridor)
		}
	}
	for _, c := range corridor {
		if IsAnyRoomWallCell(c.X, c.Y, rooms) {
			grid.set(c.X, c.Y, SymDoor)
		}
	}

	for _, r := range rooms {
		if r == nil {
			continue
		}
		for _, it := range r.Items {
			grid.set(it.X, it.Y, SymItem)
		}
	}

	if withOpponents {
		for _, o := range s.Level.AllOpponents() {
			if o.IsAlive() {
				grid.set(o.X, o.Y, o.Type.Symbol())
			}
		}
	}

	grid.set(s.Level.Exit.X, s.Level.Exit.Y, SymExit)
	grid.set(s.Player.X, s.Player.Y, SymPlayer)
	return grid
}
