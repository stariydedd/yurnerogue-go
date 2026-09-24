package domain

// CombatEvent observes a resolved attack, including lethal hits and misses.
// Coordinates belong to the instant of impact, not to an actor's later position.
// Events consume no RNG and are not part of the replay or simulation rules.
type CombatEvent struct {
	Target       Point
	Level        int
	Damage       int // Miss means no contact; zero is still a successful hit.
	TargetPlayer bool
	MaxHP        bool
	Parried      bool // the hero blocked this hit completely
	Critical     bool // a Critical Strike that landed
}

const maxCombatEvents = 64

func (s *Session) recordCombat(event CombatEvent) {
	event.Level = s.LevelNum
	// A RUN can resolve many turns before the next frame. Retain recent impacts
	// without allowing presentation data to grow for the lifetime of a run.
	if len(s.CombatEvents) == maxCombatEvents {
		copy(s.CombatEvents, s.CombatEvents[1:])
		s.CombatEvents = s.CombatEvents[:maxCombatEvents-1]
	}
	s.CombatEvents = append(s.CombatEvents, event)
}
