package domain

import "math/rand"

// OpponentType: тип врага. Отображаются как узнаваемые герои Dota.
type OpponentType int

const (
	Zombie  OpponentType = iota // Pudge
	Vampire                     // Bloodseeker
	Ghost                       // Riki
	Ogre                        // Axe
	Snake                       // Skywrath Mage
)

// AllOpponentTypes: для случайного выбора при генерации уровня.
var AllOpponentTypes = []OpponentType{Zombie, Vampire, Ghost, Ogre, Snake}

// DisplayName: имя врага в сообщениях и справке.
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

// Symbol: символ врага в сетке карты.
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

// SpriteRole: роль спрайта в assets/custom.
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

// hostility: радиус, в котором враг замечает игрока.
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

// Opponent: враг на уровне.
type Opponent struct {
	rng      *rand.Rand
	Type     OpponentType
	X, Y     int
	Health   int
	Agility  int
	Strength int

	IsVisible bool
	IsChasing bool
	Facing    int

	hostility hostility
	level     int // уровень, для которого создан враг; 0 в тестовых сценах

	// Особенности поведения отдельных типов.
	lastDirection      *Point // Snake: не повторяет прошлый диагональный шаг
	ogreCooldown       bool   // Ogre: отдыхает ход после атаки
	vampireFirstStrike bool   // Vampire: отражает первую атаку игрока
}

// baseStats: характеристики врага до масштабирования по уровню.
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
// Riki с самого начала невидим, иначе до первого хода врагов его было бы
// видно на каждом новом уровне.
func NewOpponent(t OpponentType) *Opponent {
	health, agility, strength, h := baseStats(t)
	return &Opponent{
		Type: t, Health: health, Agility: agility, Strength: strength,
		hostility: h, IsVisible: t != Ghost, Facing: 1, vampireFirstStrike: true,
	}
}

// RandomOpponent создаёт случайного врага, усиленного под номер уровня:
// каждая характеристика растёт на свой процент за уровень (см. consts.go).
func RandomOpponent(levelNum int, rng ...*rand.Rand) *Opponent {
	op := NewOpponent(pick(AllOpponentTypes, rng...))
	op.rng = source(rng)
	op.level = levelNum
	grow := func(value int, percent float64) int {
		return int(float64(value) * (1 + percent*float64(max(0, levelNum-1))/100))
	}
	op.Health = grow(op.Health, EnemyHealthGrowthPercent)
	op.Agility = grow(op.Agility, EnemyAgilityGrowthPercent)
	op.Strength = grow(op.Strength, EnemyStrengthGrowthPercent)
	return op
}

// MaxHealthDrain: сколько максимума здоровья крадёт удар Bloodseeker:
// растёт с глубиной, а не зависит от запаса здоровья игрока.
func (o *Opponent) MaxHealthDrain() int {
	return BloodseekerDrainBase + BloodseekerDrainPerLevel*max(1, o.level)
}

// IsAlive: жив ли враг.
func (o *Opponent) IsAlive() bool { return o.Health > 0 }

// TakeDamage наносит урон, здоровье не уходит ниже нуля.
func (o *Opponent) TakeDamage(damage int) {
	o.Health -= damage
	if o.Health < 0 {
		o.Health = 0
	}
}

// CanSeePlayer: попал ли игрок в радиус агрессии.
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

// Resting: Axe отдыхает ход после атаки, следующим ходом контратакует.
func (o *Opponent) Resting() bool { return o.ogreCooldown }

// SetResting ставит или снимает отдых Axe.
func (o *Opponent) SetResting(v bool) { o.ogreCooldown = v }

var (
	dirs4 = []Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}
	dirs8 = []Point{{0, -1}, {0, 1}, {-1, 0}, {1, 0}, {-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	diag4 = []Point{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
)

// shuffled возвращает перемешанную копию списка направлений.
func shuffled(src []Point, rng ...*rand.Rand) []Point {
	out := append([]Point(nil), src...)
	random(source(rng)).Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// moveArea: всё, от чего зависит ход врага: где стоят игрок и портал, где
// пол и кто ещё на уровне.
type moveArea struct {
	player, exit Point
	rooms        []*Room
	passages     []Rect
	opponents    []*Opponent
}

// canStep: можно ли врагу встать на клетку: пол комнаты или коридор, не
// занятые другим живым врагом. Клетки игрока и портала всегда заняты: враг
// бьёт игрока с соседней клетки, а портал никогда не загораживает.
func (o *Opponent) canStep(x, y int, a moveArea) bool {
	if !InBounds(x, y) || (Point{x, y}) == a.player || (Point{x, y}) == a.exit {
		return false
	}
	if !IsAnyRoomFloorCell(x, y, a.rooms) && !InPassageCenter(x, y, a.passages) {
		return false
	}
	for _, other := range a.opponents {
		if other != o && other.IsAlive() && other.X == x && other.Y == y {
			return false
		}
	}
	return true
}

// pathStep: первый шаг кратчайшего пути к игроку (обход в ширину) или nil.
func (o *Opponent) pathStep(a moveArea) *Point {
	tx, ty := a.player.X, a.player.Y
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
			if !visited[p] && o.canStep(nx, ny, a) {
				visited[p] = true
				queue = append(queue, node{nx, ny, step})
			}
		}
	}
	return nil
}

// patternStep: ход по собственному паттерну типа, когда игрок не преследуется.
func (o *Opponent) patternStep(a moveArea) *Point {
	switch o.Type {
	case Zombie:
		return o.firstWalkable(shuffled(dirs4, o.rng), a)
	case Vampire:
		return o.firstWalkable(shuffled(dirs8, o.rng), a)
	case Ghost:
		if step := o.blinkInRoom(a); step != nil {
			return step
		}
		// Прыжку нужна комната: в коридоре или в двери Riki ходит, иначе он
		// стоял бы там навсегда, почти всегда невидимый, и перекрывал проход.
		return o.firstWalkable(shuffled(dirs4, o.rng), a)
	case Ogre:
		return o.doubleStep(a)
	case Snake:
		return o.diagonalStep(a)
	}
	return nil
}

// firstWalkable возвращает первое проходимое направление из списка.
func (o *Opponent) firstWalkable(candidates []Point, a moveArea) *Point {
	for _, d := range candidates {
		if o.canStep(o.X+d.X, o.Y+d.Y, a) {
			dd := d
			return &dd
		}
	}
	return nil
}

// blinkInRoom: Riki телепортируется в случайную другую клетку своей комнаты;
// nil, если он не в комнате или свободной клетки не нашлось.
func (o *Opponent) blinkInRoom(a moveArea) *Point {
	var room *Room
	for _, r := range a.rooms {
		if r != nil && r.IsFloorCell(o.X, o.Y) {
			room = r
			break
		}
	}
	if room == nil {
		return nil
	}
	for i := 0; i < 16; i++ {
		nx := room.X + random(o.rng).Intn(room.W)
		ny := room.Y + random(o.rng).Intn(room.H)
		if (nx != o.X || ny != o.Y) && o.canStep(nx, ny, a) {
			return &Point{nx - o.X, ny - o.Y}
		}
	}
	return nil
}

// doubleStep: Axe шагает на две клетки, если свободны обе.
func (o *Opponent) doubleStep(a moveArea) *Point {
	for _, d := range shuffled(dirs4, o.rng) {
		if o.canStep(o.X+d.X, o.Y+d.Y, a) &&
			o.canStep(o.X+d.X*OgreStep, o.Y+d.Y*OgreStep, a) {
			return &Point{d.X * OgreStep, d.Y * OgreStep}
		}
	}
	return nil
}

// diagonalStep: Skywrath ходит по диагонали, стараясь не повторять прошлый шаг.
func (o *Opponent) diagonalStep(a moveArea) *Point {
	for _, d := range shuffled(diag4, o.rng) {
		if o.lastDirection != nil && *o.lastDirection == d {
			continue
		}
		if o.canStep(o.X+d.X, o.Y+d.Y, a) {
			dd := d
			o.lastDirection = &dd
			return &dd
		}
	}
	if o.lastDirection != nil {
		d := *o.lastDirection
		if o.canStep(o.X+d.X, o.Y+d.Y, a) {
			return &d
		}
	}
	return nil
}

// Move двигает врага: преследует игрока по кратчайшему пути либо идёт по
// паттерну. На клетку портала exit враг не встаёт.
func (o *Opponent) Move(px, py int, exit Point, rooms []*Room, passages []Rect, opponents []*Opponent) {
	a := moveArea{player: Point{px, py}, exit: exit, rooms: rooms, passages: passages, opponents: opponents}
	dist := abs(o.X-px) + abs(o.Y-py)

	var step *Point
	if o.CanSeePlayer(dist) {
		o.IsChasing = true
		step = o.pathStep(a)
	}
	if step == nil {
		if o.IsChasing && dist > HighHostilityRadius*2 {
			o.IsChasing = false
		}
		step = o.patternStep(a)
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
