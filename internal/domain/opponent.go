package domain

import "math/rand"

// OpponentType — тип врага. Отображаются как узнаваемые герои Dota.
type OpponentType int

const (
	Zombie  OpponentType = iota // Pudge
	Vampire                     // Bloodseeker
	Ghost                       // Riki
	Ogre                        // Axe
	Snake                       // Skywrath Mage
)

// AllOpponentTypes — для случайного выбора при генерации уровня.
var AllOpponentTypes = []OpponentType{Zombie, Vampire, Ghost, Ogre, Snake}

// DisplayName — имя врага в сообщениях и справке.
func (t OpponentType) DisplayName() string {
	switch t {
	case Zombie:
		return "Pudge"
	case Vampire:
		return "Bloodseeker"
	case Ghost:
		return "Riki"
	case Ogre:
		return "Axe"
	case Snake:
		return "Skywrath Mage"
	}
	return "Unknown"
}

// Symbol — символ врага в сетке карты.
func (t OpponentType) Symbol() byte {
	switch t {
	case Zombie:
		return SymZombie
	case Vampire:
		return SymVampire
	case Ghost:
		return SymGhost
	case Ogre:
		return SymOgre
	case Snake:
		return SymSnake
	}
	return '?'
}

// SpriteRole — роль спрайта в assets/custom.
func (t OpponentType) SpriteRole() string {
	switch t {
	case Zombie:
		return "pudge"
	case Vampire:
		return "bloodseeker"
	case Ghost:
		return "riki"
	case Ogre:
		return "axe"
	case Snake:
		return "skywrath"
	}
	return "pudge"
}

// hostility — радиус, в котором враг замечает игрока.
type hostility int

const (
	hostilityLow hostility = iota
	hostilityAverage
	hostilityHigh
)

func (h hostility) radius() int {
	switch h {
	case hostilityLow:
		return LowHostilityRadius
	case hostilityHigh:
		return HighHostilityRadius
	}
	return AverageHostilityRadius
}

// Opponent — враг на уровне.
type Opponent struct {
	Type     OpponentType
	X, Y     int
	Health   int
	Agility  int
	Strength int

	IsVisible bool
	IsChasing bool
	Facing    int

	hostility hostility

	// Особенности поведения отдельных типов.
	lastDirection      *Point // Snake: не повторяет прошлый диагональный шаг
	ogreCooldown       bool   // Ogre: отдыхает ход после атаки
	vampireFirstStrike bool   // Vampire: отражает первую атаку игрока
}

// baseStats — характеристики врага до масштабирования по уровню.
func baseStats(t OpponentType) (health, agility, strength int, h hostility) {
	switch t {
	case Zombie:
		return 50, 25, 125, hostilityAverage
	case Vampire:
		return 50, 75, 125, hostilityHigh
	case Ghost:
		return 75, 75, 25, hostilityLow
	case Ogre:
		return 150, 25, 100, hostilityAverage
	case Snake:
		return 100, 100, 30, hostilityHigh
	}
	return 10, 10, 10, hostilityAverage
}

// NewOpponent создаёт врага заданного типа (для тестов и точечных спавнов).
func NewOpponent(t OpponentType) *Opponent {
	health, agility, strength, h := baseStats(t)
	return &Opponent{
		Type: t, Health: health, Agility: agility, Strength: strength,
		hostility: h, IsVisible: true, Facing: 1, vampireFirstStrike: true,
	}
}

// RandomOpponent создаёт случайного врага, усиленного под номер уровня:
// характеристики растут на PercentsUpdateDifficultyMonsters процентов за уровень.
func RandomOpponent(levelNum int) *Opponent {
	op := NewOpponent(pick(AllOpponentTypes))
	scale := 1 + float64(PercentsUpdateDifficultyMonsters*levelNum)/100.0
	op.Health = int(float64(op.Health) * scale)
	op.Agility = int(float64(op.Agility) * scale)
	op.Strength = int(float64(op.Strength) * scale)
	return op
}

// IsAlive — жив ли враг.
func (o *Opponent) IsAlive() bool { return o.Health > 0 }

// TakeDamage наносит урон, здоровье не уходит ниже нуля.
func (o *Opponent) TakeDamage(damage int) {
	o.Health -= damage
	if o.Health < 0 {
		o.Health = 0
	}
}

// CanSeePlayer — попал ли игрок в радиус агрессии.
func (o *Opponent) CanSeePlayer(distance int) bool {
	return distance <= o.hostility.radius()
}

// DeflectsFirstStrike сообщает, отражает ли Bloodseeker первую атаку,
// и снимает этот флаг.
func (o *Opponent) DeflectsFirstStrike() bool {
	if o.Type != Vampire || !o.vampireFirstStrike {
		return false
	}
	o.vampireFirstStrike = false
	return true
}

// Resting — Axe отдыхает ход после атаки, следующим ходом контратакует.
func (o *Opponent) Resting() bool { return o.ogreCooldown }

// SetResting ставит или снимает отдых Axe.
func (o *Opponent) SetResting(v bool) { o.ogreCooldown = v }

var (
	dirs4 = []Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	dirs8 = []Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}, {-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	diag4 = []Point{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
)

// shuffled возвращает перемешанную копию списка направлений.
func shuffled(src []Point) []Point {
	out := append([]Point(nil), src...)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// canStep — можно ли врагу встать на клетку: пол комнаты или коридор,
// не занятые другим живым врагом.
func (o *Opponent) canStep(x, y int, rooms []*Room, passages []Rect, opponents []*Opponent) bool {
	if !InBounds(x, y) {
		return false
	}
	if !IsAnyRoomFloorCell(x, y, rooms) && !InPassageCenter(x, y, passages) {
		return false
	}
	for _, other := range opponents {
		if other != o && other.IsAlive() && other.X == x && other.Y == y {
			return false
		}
	}
	return true
}

// pathStep — первый шаг кратчайшего пути к цели (обход в ширину) или nil.
func (o *Opponent) pathStep(tx, ty int, rooms []*Room, passages []Rect, opponents []*Opponent) *Point {
	if o.X == tx && o.Y == ty {
		return nil
	}
	type node struct {
		x, y  int
		first *Point
	}
	queue := []node{{o.X, o.Y, nil}}
	visited := map[Point]bool{{o.X, o.Y}: true}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range dirs4 {
			nx, ny := cur.x+d.X, cur.y+d.Y
			step := cur.first
			if step == nil {
				dd := d
				step = &dd
			}
			if nx == tx && ny == ty {
				return step
			}
			p := Point{nx, ny}
			if !visited[p] && o.canStep(nx, ny, rooms, passages, opponents) {
				visited[p] = true
				queue = append(queue, node{nx, ny, step})
			}
		}
	}
	return nil
}

// patternStep — ход по собственному паттерну типа, когда игрок не преследуется.
func (o *Opponent) patternStep(rooms []*Room, passages []Rect, opponents []*Opponent) *Point {
	switch o.Type {
	case Zombie:
		return o.firstWalkable(shuffled(dirs4), rooms, passages, opponents)
	case Vampire:
		return o.firstWalkable(shuffled(dirs8), rooms, passages, opponents)
	case Ghost:
		return o.blinkInRoom(rooms, passages, opponents)
	case Ogre:
		return o.doubleStep(rooms, passages, opponents)
	case Snake:
		return o.diagonalStep(rooms, passages, opponents)
	}
	return nil
}

// firstWalkable возвращает первое проходимое направление из списка.
func (o *Opponent) firstWalkable(candidates []Point, rooms []*Room, passages []Rect, opponents []*Opponent) *Point {
	for _, d := range candidates {
		if o.canStep(o.X+d.X, o.Y+d.Y, rooms, passages, opponents) {
			dd := d
			return &dd
		}
	}
	return nil
}

// blinkInRoom — Riki телепортируется в случайную клетку своей комнаты.
func (o *Opponent) blinkInRoom(rooms []*Room, passages []Rect, opponents []*Opponent) *Point {
	var room *Room
	for _, r := range rooms {
		if r != nil && r.IsFloorCell(o.X, o.Y) {
			room = r
			break
		}
	}
	if room == nil {
		return nil
	}
	for i := 0; i < 16; i++ {
		nx := room.X + rand.Intn(room.W)
		ny := room.Y + rand.Intn(room.H)
		if o.canStep(nx, ny, rooms, passages, opponents) {
			return &Point{nx - o.X, ny - o.Y}
		}
	}
	return nil
}

// doubleStep — Axe шагает на две клетки, если свободны обе.
func (o *Opponent) doubleStep(rooms []*Room, passages []Rect, opponents []*Opponent) *Point {
	for _, d := range shuffled(dirs4) {
		if o.canStep(o.X+d.X, o.Y+d.Y, rooms, passages, opponents) &&
			o.canStep(o.X+d.X*OgreStep, o.Y+d.Y*OgreStep, rooms, passages, opponents) {
			return &Point{d.X * OgreStep, d.Y * OgreStep}
		}
	}
	return nil
}

// diagonalStep — Skywrath ходит по диагонали, стараясь не повторять прошлый шаг.
func (o *Opponent) diagonalStep(rooms []*Room, passages []Rect, opponents []*Opponent) *Point {
	for _, d := range shuffled(diag4) {
		if o.lastDirection != nil && *o.lastDirection == d {
			continue
		}
		if o.canStep(o.X+d.X, o.Y+d.Y, rooms, passages, opponents) {
			dd := d
			o.lastDirection = &dd
			return &dd
		}
	}
	if o.lastDirection != nil {
		d := *o.lastDirection
		if o.canStep(o.X+d.X, o.Y+d.Y, rooms, passages, opponents) {
			return &d
		}
	}
	return nil
}

// Move двигает врага: преследует игрока по кратчайшему пути либо идёт по паттерну.
func (o *Opponent) Move(px, py int, rooms []*Room, passages []Rect, opponents []*Opponent) {
	dist := abs(o.X-px) + abs(o.Y-py)

	var step *Point
	if o.CanSeePlayer(dist) {
		o.IsChasing = true
		step = o.pathStep(px, py, rooms, passages, opponents)
	}
	if step == nil {
		if o.IsChasing && dist > HighHostilityRadius*2 {
			o.IsChasing = false
		}
		step = o.patternStep(rooms, passages, opponents)
	}
	if step == nil {
		return
	}
	o.X += step.X
	o.Y += step.Y
	if step.X != 0 {
		o.Facing = 1
		if step.X < 0 {
			o.Facing = -1
		}
	}
}

// FacePlayer поворачивает врага к игроку (перед атакой).
func (o *Opponent) FacePlayer(px int) {
	if px > o.X {
		o.Facing = 1
	} else if px < o.X {
		o.Facing = -1
	}
}
