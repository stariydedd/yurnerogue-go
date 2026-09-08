package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// AnimFrameTicks — сколько тиков держится кадр idle-анимации (60 TPS).
const AnimFrameTicks = 10

// Architecture is anchored at the foot of the exit tile, like forest ruins.
// drawGate protects overlapping gameplay sprites instead of shrinking scenery.
func portalBounds(p domain.Point) image.Rectangle {
	x, y := p.X*TileSize+TileSize/2, (p.Y+1)*TileSize
	return image.Rect(x-28, y-64, x+28, y)
}

// cellHash — детерминированный, но хорошо перемешанный хеш клетки.
//
// Линейная формула вида x*7+y*13 даёт периодичные узоры вдоль рядов («дерево
// каждые N клеток»); битовое перемешивание убирает периодичность, сохраняя
// стабильность между кадрами.
func cellHash(x, y int) int {
	h := x*374761393 + y*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return (h ^ (h >> 16)) & 0x7FFFFFFF
}

// cameraOffset — смещение камеры в пикселях: центр на игроке, с прижатием
// к краям карты.
func cameraOffset(l Layout, px, py int) (int, int) {
	mapW, mapH := domain.Cols*TileSize, domain.Rows*TileSize
	cx := px*TileSize + TileSize/2 - l.GridW/2
	cy := py*TileSize + TileSize/2 - l.GridH/2
	return clamp(cx, 0, max(0, mapW-l.GridW)), clamp(cy, 0, max(0, mapH-l.GridH))
}

// DrawWorld рисует уровень: тайлы, предметы, врагов и туман войны.
func (r *Renderer) DrawWorld(screen *ebiten.Image, s *domain.Session) {
	l := r.Layout
	grid := s.BuildGrid(false)
	vis := s.ComputeVisibility(grid)
	camX, camY := cameraOffset(l, s.Player.X, s.Player.Y)
	tick := r.tick / AnimFrameTicks
	paths := domain.PathCells(s.Level.Rooms, s.Level.Passages)

	// Игровое поле отделено от панели: рисуем только в его пределах.
	field := screen.SubImage(image.Rect(0, 0, l.GridW, l.GridH)).(*ebiten.Image)
	field.Fill(Black)

	r.drawCachedForest(field, grid, vis, paths, forestCacheKey{
		level: s.Level, player: domain.Point{X: s.Player.X, Y: s.Player.Y}, visited: len(s.VisitedRooms),
		viewport: image.Rect(camX, camY, camX+l.GridW, camY+l.GridH),
	})
	r.drawGate(field, s, vis, camX, camY, tick)

	for _, it := range s.Level.Items {
		if vis.Visible[domain.Point{X: it.X, Y: it.Y}] {
			r.drawPickup(field, itemRole(it.Type), it.X, it.Y, camX, camY, tick)
		}
	}

	for _, actor := range worldActors(s, vis) {
		r.drawEntity(field, actor.role, actor.x, actor.y, camX, camY, tick, actor.facing)
	}
}

// isFloor — клетки, под которыми рисуется пол. Сетка помечает предметы и
// игрока своими символами, но визуально это тот же пол.
func isFloor(cell byte) bool {
	switch cell {
	case domain.SymRoomFloor, domain.SymCorridor, domain.SymDoor,
		domain.SymExit, domain.SymItem, domain.SymPlayer:
		return true
	}
	return false
}

// itemRole — роль спрайта для категории предмета.
func itemRole(t domain.ItemType) string {
	switch t {
	case domain.ItemFood:
		return "food"
	case domain.ItemElixir:
		return "elixir"
	case domain.ItemScroll:
		return "scroll"
	case domain.ItemWeapon:
		return "sword"
	}
	return "food"
}

// drawTile рисует спрайт клетки с якорем в её левом верхнем углу.
func (r *Renderer) drawTile(dst *ebiten.Image, role string, x, y, camX, camY, tick int) {
	img := r.sprites.Frame(role, tick)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x*TileSize-camX), float64(y*TileSize-camY))
	dst.DrawImage(img, op)
}

// drawEntity рисует персонажа с якорем по низу клетки: высокие спрайты
// возвышаются над тайлом.
func (r *Renderer) drawEntity(dst *ebiten.Image, role string, x, y, camX, camY, tick, facing int) {
	img, outlined := r.sprites.heroFrame(role, tick)
	if img == nil {
		return
	}
	op := heroDrawOptions(img.Bounds().Size(), x, y, camX, camY, facing, outlined)
	dst.DrawImage(img, op)
}

// dimCell затемняет клетку, которую игрок помнит, но сейчас не видит.
func (r *Renderer) dimCell(dst *ebiten.Image, x, y, camX, camY int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x*TileSize-camX), float64(y*TileSize-camY))
	dst.DrawImage(r.dim, op)
}
