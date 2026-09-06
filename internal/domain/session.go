package domain

import "math/rand"

// Stats — статистика забега, уходит в лидерборд по его завершении.
type Stats struct {
	EnemiesKilled int
	FoodUsed      int
	ElixirsUsed   int
	ScrollsRead   int
	AttacksMade   int
	HitsTaken     int
	TilesMoved    int
}

// Session — игровая сессия: текущий уровень, персонаж и статистика.
type Session struct {
	actions        []byte
	Turns          int
	ReplayOverflow bool
	LevelNum       int
	Level          *Level
	Player         *Person
	Message        string
	Stats          Stats

	// VisitedRooms — индексы комнат, которые игрок уже видел (для тумана войны).
	VisitedRooms map[int]bool
}

// NewSession начинает забег с первого уровня.
func NewSession() *Session { return NewSessionAtLevel(1) }

// NewSessionAtLevel начинает сессию с заданного уровня (удобно в тестах).
func NewSessionAtLevel(num int) *Session {
	return newSessionAtLevel(num, rand.New(rand.NewSource(rand.Int63())))
}

func NewSessionSeed(seed int64) *Session {
	return newSessionAtLevel(1, rand.New(rand.NewSource(seed)))
}

func newSessionAtLevel(num int, rng *rand.Rand) *Session {
	level := NewLevel(num, rng)
	start := level.PlayerStart()

	player := NewPerson()
	player.rng = rng
	player.X, player.Y = start.X, start.Y
	level.GenerateItems(player)

	return &Session{
		LevelNum:     num,
		Level:        level,
		Player:       player,
		VisitedRooms: map[int]bool{},
	}
}

// SetMessage кладёт текст в строку сообщений HUD.
func (s *Session) SetMessage(msg string) { s.Message = msg }

// Exit — координаты выхода текущего уровня.
func (s *Session) Exit() Point { return s.Level.Exit }

// Opponents — все враги текущего уровня.
func (s *Session) Opponents() []*Opponent { return s.Level.AllOpponents() }

// OpponentAt — живой враг в клетке или nil.
func (s *Session) OpponentAt(x, y int) *Opponent {
	for _, o := range s.Level.AllOpponents() {
		if o.IsAlive() && o.X == x && o.Y == y {
			return o
		}
	}
	return nil
}

// UpdateLevel спускает игрока на следующий уровень.
func (s *Session) UpdateLevel() {
	s.LevelNum++
	s.Level = NewLevel(s.LevelNum, s.Player.rng)
	start := s.Level.PlayerStart()
	s.Player.X, s.Player.Y = start.X, start.Y
	s.Level.GenerateItems(s.Player)
	s.VisitedRooms = map[int]bool{}
}

// Won — забег завершён победой: спуск после последнего уровня.
func (s *Session) Won() bool { return s.LevelNum > MaxLevels }
