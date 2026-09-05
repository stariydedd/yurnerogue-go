package game

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestSkywrathSleepSkipsNextAction(t *testing.T) {
	for _, initiallySleeping := range []bool{false, true} {
		name := "new sleep"
		if initiallySleeping {
			name = "reapplied sleep"
		}
		t.Run(name, func(t *testing.T) {
			p := domain.NewPerson()
			room := &domain.Room{X: 4, Y: 4, W: 8, H: 8}
			s := &domain.Session{Player: p, LevelNum: 1, Level: &domain.Level{
				Rooms: []*domain.Room{room}, Exit: domain.Point{X: 11, Y: 11},
			}}
			mage := domain.NewOpponent(domain.Snake)
			mage.X, mage.Y = 6, 6
			mage.Agility = 1_000_000 // Всегда попадает, но не убивает игрока.
			mage.Strength = -1_000
			room.Enemies = []*domain.Opponent{mage}
			g := &Game{session: s, state: StatePlaying}

			// Усыпление имеет шанс 15%; ждём именно срабатывания эффекта.
			// При 1000 попытках вероятность не дождаться меньше 10^-70.
			cast := false
			for attempt := 0; attempt < 1000; attempt++ {
				p.X, p.Y = 5, 5
				if initiallySleeping {
					p.FallAsleep(1)
				}
				g.HandleKey(ebiten.KeyD)
				// Нулевой урон не включает текст сна в attackMessage, поэтому
				// проверяем сам эффект; после первой фазы он должен жить один ход.
				if p.Sleeping {
					cast = true
					break
				}
			}
			if !cast {
				t.Fatal("Skywrath sleep never survived the enemy phase")
			}
			if p.SleepTurns != 1 {
				t.Fatalf("sleep lasts %d turns, want 1", p.SleepTurns)
			}

			room.Enemies = nil // Следующий ход не накладывает новый сон.
			x, y := p.X, p.Y
			g.HandleKey(ebiten.KeyD)
			if p.X != x || p.Y != y || p.Sleeping {
				t.Fatal("next action must be skipped and consume the sleep")
			}
			if !strings.Contains(s.Message, "asleep") {
				t.Fatalf("missing sleep feedback: %q", s.Message)
			}
			g.HandleKey(ebiten.KeyD)
			if p.X != x+1 || p.Y != y {
				t.Fatal("movement must resume after one skipped action")
			}
		})
	}
}
