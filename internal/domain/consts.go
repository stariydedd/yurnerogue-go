// Package domain содержит правила и состояние игры: генерацию уровней, бой,
// движение, предметы. Не зависит ни от отрисовки, ни от способа хранения —
// ровно как domain/ в Python-версии.
package domain

// Размеры карты и сетки размещения комнат.
const (
	Rows = 45
	Cols = 100

	GridDim = 3 // комнаты раскладываются по сетке GridDim x GridDim
	CellW   = Cols / GridDim
	CellH   = Rows / GridDim

	ProbRoom      = 0.75 // вероятность, что ячейка сетки получит комнату
	MinW, MaxW    = 4, 16
	MinH, MaxH    = 3, 12
	Pad           = 2
	MinRoomsCount = 4
)

// Правила забега.
const (
	MaxLevels               = 21
	MaxBackpackItemsPerType = 9
	MaxWeaponBonus          = 50
	ElixirDuration          = 20
	MaxMonstersPerRoom      = 2
	MaxConsumablesPerRoom   = 3
	LevelUpdateDifficulty   = 10
)

// Символы клеток карты.
const (
	SymEmpty     byte = ' '
	SymWall      byte = '#'
	SymRoomFloor byte = '.'
	SymCorridor  byte = '*'
	SymDoor      byte = '|'
	SymPlayer    byte = '@'
	SymExit      byte = 'E'
	SymItem      byte = 'I'
)

// Символы врагов на карте.
const (
	SymZombie  byte = 'z'
	SymVampire byte = 'v'
	SymGhost   byte = 'g'
	SymOgre    byte = 'O'
	SymSnake   byte = 's'
)

// Walkable сообщает, можно ли встать на клетку с таким символом.
func Walkable(sym byte) bool {
	switch sym {
	case SymRoomFloor, SymCorridor, SymDoor, SymExit, SymItem:
		return true
	}
	return false
}

// Стартовые характеристики игрока.
const (
	DefaultMaxHealth = 500
	DefaultAgility   = 70
	DefaultStrength  = 70
	DefaultTreasures = 0
)

// Формулы боя.
const (
	InitialHitChance = 70
	StandardAgility  = 50
	AgilityFactor    = 0.3
	InitialDamage    = 30
	StandardStrength = 50
	StrengthFactor   = 0.3
	StrengthAddition = 65
	// Critical Strike recharges with ordinary attacks, not with turns, so walking
	// between fights does not make every fight start with it.
	StrikeRechargeAttacks = 3
	// Parry needs an enemy in contact. It recharges with enemy hits
	// taken outside the parry turn, blocks incoming hits completely and strikes back
	// every attacker for this share of a hit.
	GuardRechargeHits    = 3
	RiposteDamagePercent = 50

	// Bloodseeker drains max HP: base + per-level amount, fully blocked by a parry.
	BloodseekerDrainBase     = 10
	BloodseekerDrainPerLevel = 2
	SleepChance              = 15
	ChanceGhostVisible       = 20
	OgreStep                 = 2

	// Enemy stats grow per level by these percents. Health and strength grow
	// fastest; agility changes hit chances directly, so it grows slowly.
	EnemyHealthGrowthPercent   = 8.0
	EnemyStrengthGrowthPercent = 4.0
	EnemyAgilityGrowthPercent  = 1.5
)

// Радиусы агрессии врагов.
const (
	LowHostilityRadius     = 2
	AverageHostilityRadius = 4
	HighHostilityRadius    = 6
)

// PlayerName — имя героя в HUD и сообщениях.
const PlayerName = "Juggernaut"
