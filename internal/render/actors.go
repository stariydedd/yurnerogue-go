package render

import (
	"sort"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

type worldActor struct {
	role         string
	x, y, facing int
}

// Sort a render-only list by the ground anchor, not the top of the artwork.
// All feet use (y+1)*TileSize, so row order is depth order regardless of sprite
// height, outline padding, camera position or animation. Equal rows preserve
// the existing enemy order, with the player last, to avoid arbitrary flicker.
func worldActors(s *domain.Session, vis domain.Visibility) []worldActor {
	enemies := s.Level.AllOpponents()
	actors := make([]worldActor, 0, len(enemies)+1)
	for _, op := range enemies {
		if !op.IsAlive() || !op.IsVisible || !vis.Visible[domain.Point{X: op.X, Y: op.Y}] {
			continue
		}
		actors = append(actors, worldActor{role: op.Type.SpriteRole(), x: op.X, y: op.Y, facing: op.Facing})
	}
	actors = append(actors, worldActor{role: "player", x: s.Player.X, y: s.Player.Y, facing: s.Player.Facing})
	sort.SliceStable(actors, func(i, j int) bool { return actors[i].y < actors[j].y })
	return actors
}
