package domain

// RunLimit — предохранитель от бесконечного бега.
const RunLimit = 200

// CanMoveTo — можно ли игроку встать на клетку. Проходимы пол комнат и
// центральные линии коридоров (двери и выход лежат на них же).
//
// Считается по геометрии напрямую: Python-версия ради каждой такой проверки
// собирала всю карту символов заново.
func (s *Session) CanMoveTo(x, y int) bool {
	if !InBounds(x, y) {
		return false
	}
	return IsAnyRoomFloorCell(x, y, s.Level.Rooms) || InPassageCenter(x, y, s.Level.Passages)
}

// MoveX — шаг игрока по горизонтали; если в клетке враг, бьёт его.
// Возвращает true, если ход состоялся (шаг или атака).
func (s *Session) MoveX(dir int) bool {
	// Разворот происходит даже если шаг упёрся в стену или врага.
	s.Player.Facing = 1
	if dir < 0 {
		s.Player.Facing = -1
	}
	return s.step(s.Player.X+dir, s.Player.Y)
}

// MoveY — шаг игрока по вертикали; если в клетке враг, бьёт его.
func (s *Session) MoveY(dir int) bool {
	return s.step(s.Player.X, s.Player.Y+dir)
}

// step — общий код шага: атака врага в целевой клетке либо перемещение.
func (s *Session) step(nx, ny int) bool {
	if op := s.OpponentAt(nx, ny); op != nil {
		s.attack(op)
		return true
	}
	if s.CanMoveTo(nx, ny) {
		s.Player.X, s.Player.Y = nx, ny
		s.Stats.TilesMoved++
		return true
	}
	s.SetMessage("Can't move that way.")
	return false
}

// attack — удар игрока по врагу с записью статистики и сообщением.
func (s *Session) attack(op *Opponent) {
	goldBefore := s.Player.Treasures
	damage := PlayerAttacks(s.Player, op)
	s.Stats.AttacksMade++
	if !op.IsAlive() {
		s.Stats.EnemiesKilled++
	}
	s.SetMessage(playerAttackMessage(op, damage, s.Player.Treasures-goldBefore))
}

// playerAttackMessage — строка в HUD по результату удара игрока.
func playerAttackMessage(op *Opponent, damage, goldGained int) string {
	name := op.Type.DisplayName()
	if damage == Miss {
		if op.Type == Vampire {
			return "Your first strike against the " + name + " was deflected!"
		}
		return "You missed the " + name + "."
	}
	if !op.IsAlive() {
		return "You killed the " + name + " for " + itoa(damage) + " dmg! Gained " + itoa(goldGained) + " gold."
	}
	return "You hit the " + name + " for " + itoa(damage) + " dmg."
}

// CheckItemPickup подбирает предмет под игроком, если он там есть.
func (s *Session) CheckItemPickup() {
	for _, room := range s.Level.Rooms {
		if room == nil {
			continue
		}
		for i, it := range room.Items {
			if it.X != s.Player.X || it.Y != s.Player.Y {
				continue
			}
			if s.Player.PickUpItem(it) {
				room.Items = append(room.Items[:i], room.Items[i+1:]...)
				s.SetMessage("Picked up: " + it.Name + it.StatLabel() + ".")
			} else {
				s.SetMessage("Backpack full! Cannot pick up " + it.Name + ".")
			}
			return
		}
	}
}

// CheckExit спускает игрока на следующий уровень, если он встал на портал.
func (s *Session) CheckExit() bool {
	if s.Player.X != s.Level.Exit.X || s.Player.Y != s.Level.Exit.Y {
		return false
	}
	next := s.LevelNum + 1
	s.UpdateLevel()
	s.SetMessage("You descend to level " + itoa(next) + ".")
	return true
}

// ResolveTurn завершает ход игрока: подбор предмета, спуск, ходы врагов.
func (s *Session) ResolveTurn() {
	s.CheckItemPickup()
	s.CheckExit()
	s.ProcessEnemyTurns()
}

// DropItemNearPlayer кладёт предмет на ближайшую свободную клетку пола.
func (s *Session) DropItemNearPlayer(item *Item) {
	room := s.Level.RoomAt(s.Player.X, s.Player.Y)
	if room == nil {
		return
	}
	for radius := 0; radius < max(room.W, room.H); radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				x, y := s.Player.X+dx, s.Player.Y+dy
				if x == s.Player.X && y == s.Player.Y {
					continue
				}
				if !room.IsFloorCell(x, y) || room.itemAt(Point{x, y}) != nil {
					continue
				}
				if x == s.Level.Exit.X && y == s.Level.Exit.Y {
					continue
				}
				item.X, item.Y = x, y
				room.Items = append(room.Items, item)
				return
			}
		}
	}
}

// --- Бег (find из оригинального Rogue) ---

// corridorTurn — новое направление на повороте коридора или nil.
// Поворот выполняется, только если игрок стоит на тропе и ровно одно
// направление (кроме обратного) продолжает её: развилки останавливают бег.
func (s *Session) corridorTurn(dx, dy int) *Point {
	rooms, passages := s.Level.Rooms, s.Level.Passages
	if !IsCorridorCell(s.Player.X, s.Player.Y, rooms, passages) {
		return nil
	}
	var options []Point
	for _, d := range dirs4 {
		if d.X == -dx && d.Y == -dy {
			continue
		}
		tx, ty := s.Player.X+d.X, s.Player.Y+d.Y
		if IsCorridorCell(tx, ty, rooms, passages) && s.CanMoveTo(tx, ty) {
			options = append(options, d)
		}
	}
	if len(options) != 1 {
		return nil
	}
	return &options[0]
}

// doorBeside — есть ли дверь сбоку от направления движения: игрок пробегает
// мимо проёма.
func (s *Session) doorBeside(dx, dy int) bool {
	x, y := s.Player.X, s.Player.Y
	var sides [2]Point
	if dx != 0 {
		sides = [2]Point{{x, y - 1}, {x, y + 1}}
	} else {
		sides = [2]Point{{x - 1, y}, {x + 1, y}}
	}
	return s.Level.Doors[sides[0]] || s.Level.Doors[sides[1]]
}

// Run — бег в направлении до упора; каждый шаг является полноценным ходом.
//
// Бег по комнате останавливается на клетке перед дверью впереди и у двери,
// мимо которой пробегает; но если игрок уже стоит вплотную к двери, шаг в её
// сторону проносит через дверь и дальше по коридору. Коридорный бег следует
// поворотам и заканчивается в дверном проёме на том конце. Также стоп: стена
// или развилка, враг, портал впереди (на бегу не спускаемся), полученный урон,
// подобранный предмет, сон или смерть.
func (s *Session) Run(dx, dy int) {
	p := s.Player
	for step := 0; step < RunLimit; step++ {
		if !p.IsAlive() || p.Sleeping {
			return
		}
		nx, ny := p.X+dx, p.Y+dy

		if s.OpponentAt(nx, ny) != nil {
			if step == 0 {
				// Нажатие не должно выглядеть мёртвым: бег к врагу вплотную
				// не начинается, но игроку сообщается почему.
				s.SetMessage("An enemy is in the way.")
			}
			return
		}
		if nx == s.Level.Exit.X && ny == s.Level.Exit.Y {
			s.SetMessage("You stop at the portal.")
			return
		}

		enteringDoor := s.Level.Doors[Point{nx, ny}]
		fromRoom := IsAnyRoomFloorCell(p.X, p.Y, s.Level.Rooms)
		if enteringDoor && fromRoom && step > 0 {
			return // разбежались по комнате — стоп перед дверью, не в ней
		}

		if !s.CanMoveTo(nx, ny) {
			// Поворот коридора — только когда уже бежим: нажатие в сторону
			// стены не должно начинать бег вбок.
			if step > 0 {
				if turn := s.corridorTurn(dx, dy); turn != nil {
					dx, dy = turn.X, turn.Y
					continue
				}
			} else {
				s.SetMessage("Can't move that way.")
			}
			return
		}

		hpBefore := p.Health
		itemsBefore := len(p.Backpack)
		levelBefore := s.LevelNum

		var moved bool
		if dx != 0 {
			moved = s.MoveX(dx)
		} else {
			moved = s.MoveY(dy)
		}
		if !moved {
			return
		}
		s.ResolveTurn()

		if p.Health < hpBefore || len(p.Backpack) != itemsBefore || s.LevelNum != levelBefore {
			return
		}
		if enteringDoor && !fromRoom {
			return // прибежали по коридору в дверной проём — конец коридора
		}
		// Стоп у бокового проёма — только в комнате: в коридоре дверь сбоку
		// от поворота это его собственное продолжение, и бег должен доехать
		// до проёма, а не встать на углу.
		if IsAnyRoomFloorCell(p.X, p.Y, s.Level.Rooms) && s.doorBeside(dx, dy) {
			return
		}
	}
}
