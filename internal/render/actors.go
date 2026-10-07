package render

import (
	"sort"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

type worldActor struct {
	role         string
	x, y, facing int
	opponent     *domain.Opponent
}

// Сортируем список только для отрисовки по точке опоры на земле, а не по верху картинки.
// Все ноги стоят на (y+1)*TileSize, поэтому порядок строк это порядок глубины независимо от
// высоты спрайта, поля обводки, позиции камеры и анимации. Равные строки сохраняют
// прежний порядок врагов, игрок последним, чтобы не было случайного мерцания.
func worldActors(s *domain.Session, vis domain.Visibility) []worldActor {
	enemies := s.Level.AllOpponents()
	actors := make([]worldActor, 0, len(enemies)+1)
	for _, op := range enemies {
		if !op.IsAlive() || !op.IsVisible || !vis.Visible[domain.Point{X: op.X, Y: op.Y}] {
			continue
		}
		actors = append(actors, worldActor{role: op.Type.SpriteRole(), x: op.X, y: op.Y, facing: op.Facing, opponent: op})
	}
	actors = append(actors, worldActor{role: "player", x: s.Player.X, y: s.Player.Y, facing: s.Player.Facing})
	sort.SliceStable(actors, func(i, j int) bool { return actors[i].y < actors[j].y })
	return actors
}
