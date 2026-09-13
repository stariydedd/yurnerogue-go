package game

import (
	"reflect"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/sound"
)

func TestStepSurfaceMatchesRenderedPathGeometry(t *testing.T) {
	s := domain.NewSessionSeed(21)
	paths := domain.PathCells(s.Level.Rooms, s.Level.Passages)
	seen := map[sound.Cue]bool{}
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			if !s.CanMoveTo(x, y) {
				continue
			}
			s.Player.X, s.Player.Y = x, y
			want := sound.StepGrass
			if paths[domain.Point{X: x, Y: y}] {
				want = sound.StepTrail
			}
			if got := captureActionAudio(s).stepCue; got != want {
				t.Fatalf("wrong surface at %d,%d: %d instead of %d", x, y, got, want)
			}
			seen[want] = true
		}
	}
	if len(seen) != 2 {
		t.Fatal("fixture does not cover both surfaces")
	}
}

func TestFootstepsFollowMovementNotInput(t *testing.T) {
	before := actionAudioSnapshot{level: 1, position: domain.Point{X: 3, Y: 3}, stepCue: sound.StepGrass}
	for _, cue := range []sound.Cue{sound.StepGrass, sound.StepTrail} {
		after := before
		after.position.X++
		after.stats.TilesMoved++
		after.stepCue = cue
		if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{cue}) {
			t.Fatalf("missing destination surface step: %v", got)
		}
		after.stats.TilesMoved = 20
		if got := actionCues(before, after, "D"); !reflect.DeepEqual(got, []sound.Cue{cue}) {
			t.Fatal("instant RUN queued a burst of footsteps")
		}
	}
	for _, action := range []string{"w", "D", "z", "h0"} {
		if got := actionCues(before, before, action); len(got) != 0 {
			t.Fatalf("stationary action %s produced sounds: %v", action, got)
		}
	}
	after := before
	after.position.X += 5 // A position change alone is not a walking event.
	if got := actionCues(before, after, "e0"); len(got) != 0 {
		t.Fatal("teleport sounded like walking")
	}
	after.stats.TilesMoved++
	after.level++
	if got := actionCues(before, after, "d"); !reflect.DeepEqual(got, []sound.Cue{sound.Portal}) {
		t.Fatal("level transition produced a destination footstep")
	}
}

func TestRealStepsChangeSurfaceAtDoorAndStaySilentAtWall(t *testing.T) {
	s := domain.NewSessionSeed(21)
	s.Level.Rooms = []*domain.Room{{X: 3, Y: 3, W: 3, H: 3}}
	s.Level.Passages = []domain.Rect{{X: 4, Y: 3, W: 7, H: 3}}
	s.Level.Doors = domain.DoorCells(s.Level.Rooms, s.Level.Passages)
	s.Level.Items = nil
	s.Level.Exit = domain.Point{X: 40, Y: 40}
	s.Player.X, s.Player.Y = 4, 4
	for _, tc := range []struct {
		action string
		want   []sound.Cue
	}{
		{"d", []sound.Cue{sound.StepGrass}},
		{"d", []sound.Cue{sound.StepTrail}}, // Doorway, not a wooden door.
		{"d", []sound.Cue{sound.StepTrail}},
		{"w", nil}, // Corridor wall.
		{"a", []sound.Cue{sound.StepTrail}},
		{"a", []sound.Cue{sound.StepGrass}},
	} {
		before := captureActionAudio(s)
		if err := s.ApplyAction(tc.action); err != nil {
			t.Fatal(err)
		}
		if got := actionCues(before, captureActionAudio(s), tc.action); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("action %s: got %v, want %v", tc.action, got, tc.want)
		}
	}
}
