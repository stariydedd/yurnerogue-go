package render

import (
	"reflect"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestWorldActorsDepthAndStableTies(t *testing.T) {
	for _, row := range []int{9, 10, 11} {
		s := domain.NewSessionSeed(21)
		s.Player.X, s.Player.Y, s.Player.Facing = 16, 10, -1
		op := domain.NewOpponent(domain.Snake)
		op.X, op.Y, op.Facing, op.IsVisible = 16, row, 1, true
		s.Level.Rooms = []*domain.Room{{Enemies: []*domain.Opponent{op}}}
		vis := domain.Visibility{Visible: map[domain.Point]bool{{X: 16, Y: row}: true}}
		got := worldActors(s, vis)
		player := worldActor{role: "player", x: 16, y: 10, facing: -1}
		sky := worldActor{role: "skywrath", x: 16, y: row, facing: 1}
		want := []worldActor{sky, player}
		if row > 10 {
			want = []worldActor{player, sky}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("row %d: got %v, want %v", row, got, want)
		}
	}
}

func TestWorldActorsSortAllEnemiesWithoutChangingSimulation(t *testing.T) {
	s := domain.NewSessionSeed(21)
	s.Player.X, s.Player.Y = 16, 10
	vis := domain.Visibility{Visible: map[domain.Point]bool{}}
	var enemies []*domain.Opponent
	for i, kind := range domain.AllOpponentTypes {
		op := domain.NewOpponent(kind)
		op.X, op.Y, op.IsVisible = 12+i, []int{12, 9, 9, 11, 8}[i], true
		vis.Visible[domain.Point{X: op.X, Y: op.Y}] = true
		enemies = append(enemies, op)
	}
	s.Level.Rooms = []*domain.Room{{Enemies: enemies[:2]}, {Enemies: enemies[2:]}}
	before := append([]*domain.Opponent(nil), s.Level.AllOpponents()...)
	got := worldActors(s, vis)
	var roles []string
	for _, actor := range got {
		roles = append(roles, actor.role)
	}
	want := []string{"skywrath", "bloodseeker", "riki", "player", "axe", "pudge"}
	if !reflect.DeepEqual(roles, want) {
		t.Fatalf("got %v, want %v", roles, want)
	}
	if !reflect.DeepEqual(before, s.Level.AllOpponents()) {
		t.Fatal("changed gameplay enemy order")
	}
	for i := 0; i < 10; i++ {
		if !reflect.DeepEqual(got, worldActors(s, vis)) {
			t.Fatal("unstable render order")
		}
	}
	// Rebuild after movement: the old draw order must not be cached.
	enemies[0].Y = 7
	vis.Visible[domain.Point{X: enemies[0].X, Y: 7}] = true
	if worldActors(s, vis)[0].role != "pudge" {
		t.Fatal("stale depth after movement")
	}
}

func TestWorldActorsKeepVisibilityAndAliveFilters(t *testing.T) {
	s := domain.NewSessionSeed(21)
	vis := domain.Visibility{Visible: map[domain.Point]bool{}, Explored: map[domain.Point]bool{}}
	room := &domain.Room{}
	s.Level.Rooms = []*domain.Room{room}
	for i := 0; i < 3; i++ {
		op := domain.NewOpponent(domain.Snake)
		op.X, op.Y, op.IsVisible = 12+i, 10, true
		p := domain.Point{X: op.X, Y: op.Y}
		vis.Visible[p] = true
		switch i {
		case 0:
			op.Health = 0
		case 1:
			op.IsVisible = false
		case 2:
			delete(vis.Visible, p)
			vis.Explored[p] = true
		}
		room.Enemies = append(room.Enemies, op)
	}
	got := worldActors(s, vis)
	if len(got) != 1 || got[0].role != "player" {
		t.Fatalf("hidden or dead enemy rendered: %v", got)
	}
	room.Enemies = nil
	if got := worldActors(s, domain.Visibility{}); len(got) != 1 || got[0].role != "player" {
		t.Fatal("player lost in empty scene")
	}
}
