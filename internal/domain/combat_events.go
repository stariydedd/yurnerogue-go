package domain

// CombatEvent фиксирует завершённую атаку, включая смертельные удары и промахи.
// Координаты относятся к моменту удара, а не к позиции участника после него.
// События не тратят RNG и не входят ни в повтор забега, ни в правила симуляции.
type CombatEvent struct {
	Target       Point
	Level        int
	Damage       int // Miss: контакта не было; ноль всё равно успешный удар.
	TargetPlayer bool
	MaxHP        bool
	Parried      bool // герой полностью заблокировал этот удар
	Critical     bool // критический удар, который попал
	Shielded     bool // щит Clarity принял весь удар
}

const maxCombatEvents = 64

func (s *Session) recordCombat(event CombatEvent) {
	event.Level = s.LevelNum
	// Бег может пройти много ходов до следующего кадра. Последние удары храним,
	// но не даём данным для отображения расти всё время забега.
	if len(s.CombatEvents) == maxCombatEvents {
		copy(s.CombatEvents, s.CombatEvents[1:])
		s.CombatEvents = s.CombatEvents[:maxCombatEvents-1]
	}
	s.CombatEvents = append(s.CombatEvents, event)
}
