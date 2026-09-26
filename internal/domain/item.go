package domain

import "math/rand"

// BaseWeaponName is the permanent starter, represented by no equipped upgrade.
const BaseWeaponName = "Quelling Blade"

// ItemType — категория предмета.
type ItemType int

const (
	ItemNone ItemType = iota
	ItemTreasure
	ItemFood
	ItemElixir
	ItemScroll
	ItemWeapon
)

// ItemSubType — какую характеристику меняет эликсир или свиток.
type ItemSubType int

const (
	SubNone ItemSubType = iota
	SubStrength
	SubAgility
	SubHealth
)

// Item — предмет: еда, эликсир, свиток, оружие или сокровище.
type Item struct {
	Type    ItemType
	SubType ItemSubType
	Name    string

	HealthEffect    int
	MaxHealthEffect int
	AgilityEffect   int
	StrengthEffect  int

	X, Y int
}

var (
	foodNames = []string{
		"Tango",
		"Iron Branch",
		"Faerie Fire",
		"Mango",
		"Healing Salve",
		"Lotus",
		"Cheese",
		"Seeds of Serenity",
		"Elixir",
	}
	elixirNames = []string{
		"Phantom Clarity",
		"Arcane Clarity",
		"Frozen Clarity",
		"Crimson Clarity",
		"Jade Clarity",
		"Moon Clarity",
		"Mystic Clarity",
		"Ember Clarity",
		"Wind Clarity",
	}
	scrollNames = []string{
		"Aghanim's Scroll",
		"Ogre Scroll",
		"Elven Scroll",
		"Vital Scroll",
		"Arcane Scroll",
		"Sage Scroll",
		"Power Scroll",
		"Mystic Scroll",
		"Ancient Scroll",
	}
	// weaponNames go from cheapest to most expensive Dota 2 item (approximate
	// gold cost); each name covers an equal slice of the +1..+50 bonus range.
	weaponNames = []string{
		"Crystalys",      // ~2000
		"Yasha",          // ~2050
		"Diffusal Blade", // ~2500
		"Shadow Blade",   // ~3000
		"Desolator",      // ~3500
		"Battle Fury",    // ~4100
		"Radiance",       // ~4700
		"Butterfly",      // ~5000
		"Silver Edge",    // ~5500
		"Abyssal Blade",  // ~6250
	}
)

// pick возвращает случайный элемент списка.
func pick[T any](xs []T, rng ...*rand.Rand) T {
	return xs[random(source(rng)).Intn(len(xs))]
}

// rollBetween возвращает случайное число от low до high включительно.
func rollBetween(low, high int, rng ...*rand.Rand) int {
	low = max(1, low)
	high = max(low, high)
	return low + random(source(rng)).Intn(high-low+1)
}

// RandomItem создаёт случайный предмет уровня level: еду, эликсир, свиток или
// оружие. Сила находок зависит от глубины, еда от здоровья игрока.
func RandomItem(player *Person, level int) *Item {
	switch random(player.rng).Intn(4) {
	case 0:
		return NewFood(player, level)
	case 1:
		return NewElixir(player, level)
	case 2:
		return NewScroll(player, level)
	default:
		return NewWeapon(level, player.rng)
	}
}

// NewFood: еда уровня level. Лечение зависит от глубины, а не от максимума
// здоровья: Bloodseeker уменьшает максимум, а еда должна оставаться полезной.
func NewFood(player *Person, level int) *Item {
	low, high := FoodHealRange(level)
	return &Item{
		Type:         ItemFood,
		Name:         pick(foodNames, player.rng),
		HealthEffect: rollBetween(low, high, player.rng),
	}
}

// FoodHealRange: сколько лечит еда на уровне: от 25..50 на первом до
// 75..150 на последнем. Лечение не поднимает здоровье выше максимума.
func FoodHealRange(level int) (low, high int) {
	high = 45 + 5*max(1, level)
	return high / 2, high
}

// NewElixir: временный бафф на ElixirDuration ходов, вдвое сильнее свитка
// той же глубины.
func NewElixir(player *Person, level int) *Item {
	it := &Item{Type: ItemElixir, Name: pick(elixirNames, player.rng)}
	applyStatRoll(it, player, level, 2)
	return it
}

// NewScroll — постоянный бафф к одной из характеристик.
func NewScroll(player *Person, level int) *Item {
	it := &Item{Type: ItemScroll, Name: pick(scrollNames, player.rng)}
	applyStatRoll(it, player, level, 1)
	return it
}

// StatBonus: диапазон бонуса свитка на уровне: сила или ловкость от
// +2..+3 на первом до +6..+9 на последнем, максимум здоровья от половины
// предела до предела (+13..+25 на первом, +63..+125 на последнем).
// Зелья умножают обе границы на два, поэтому +1 не выпадает никогда.
type StatBonus struct {
	StatLow, StatHigh, HealthLow, HealthHigh int
}

func StatBonusRange(level int) StatBonus {
	level = max(1, level)
	healthHigh := 20 + 5*level
	return StatBonus{
		StatLow: 2 + (level-1)/5, StatHigh: 2 + (level+2)/3,
		HealthLow: (healthHigh + 1) / 2, HealthHigh: healthHigh,
	}
}

// applyStatRoll выбирает характеристику и величину бонуса из диапазона
// глубины, умноженного на multiplier.
func applyStatRoll(it *Item, player *Person, level, multiplier int) {
	b := StatBonusRange(level)
	switch random(player.rng).Intn(3) {
	case 0:
		it.SubType = SubHealth
		it.MaxHealthEffect = rollBetween(b.HealthLow*multiplier, b.HealthHigh*multiplier, player.rng)
	case 1:
		it.SubType = SubAgility
		it.AgilityEffect = rollBetween(b.StatLow*multiplier, b.StatHigh*multiplier, player.rng)
	default:
		it.SubType = SubStrength
		it.StrengthEffect = rollBetween(b.StatLow*multiplier, b.StatHigh*multiplier, player.rng)
	}
}

// WeaponBonusRange: бонус оружия на уровне: от +1..+17 на первом до
// +36..+50 на последнем. Заточка может поднять бонус выше.
func WeaponBonusRange(level int) (low, high int) {
	level = max(1, level)
	return 1 + 7*(level-1)/4, min(MaxWeaponBonus, 17+2*(level-1))
}

// NewWeapon: оружие уровня level со случайным бонусом из WeaponBonusRange;
// название определяется бонусом.
func NewWeapon(level int, rng ...*rand.Rand) *Item {
	low, high := WeaponBonusRange(level)
	bonus := low + random(source(rng)).Intn(high-low+1)
	return &Item{Type: ItemWeapon, Name: WeaponName(bonus), StrengthEffect: bonus}
}

// WeaponName: название оружия по бонусу: чем сильнее, тем дороже предмет.
func WeaponName(bonus int) string {
	step := MaxWeaponBonus / len(weaponNames)
	return weaponNames[clamp((bonus-1)/step, 0, len(weaponNames)-1)]
}

// StatLabel — приписка к сообщению об использовании предмета, например " [+3 STR]".
// У оружия приписки нет: его сила видна по названию.
func (it *Item) StatLabel() string {
	if it.Type == ItemWeapon {
		return ""
	}
	switch {
	case it.HealthEffect > 0:
		return " [+" + itoa(it.HealthEffect) + " HP]"
	case it.MaxHealthEffect > 0 && it.Type == ItemElixir:
		return " [+" + itoa(it.MaxHealthEffect) + " SHIELD]"
	case it.MaxHealthEffect > 0:
		return " [+" + itoa(it.MaxHealthEffect) + " MAX HP]"
	case it.AgilityEffect > 0:
		return " [+" + itoa(it.AgilityEffect) + " AGI]"
	case it.StrengthEffect > 0:
		return " [+" + itoa(it.StrengthEffect) + " STR]"
	}
	return ""
}
