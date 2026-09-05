package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// AnimFrameTicks — сколько тиков держится кадр idle-анимации (60 TPS).
const AnimFrameTicks = 10

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

	x0 := max(0, camX/TileSize-1)
	y0 := max(0, camY/TileSize-1)
	x1 := min(domain.Cols, (camX+l.GridW)/TileSize+2)
	y1 := min(domain.Rows, (camY+l.GridH)/TileSize+2)

	// Деревья рисуются после тайлов, чтобы кроны не резались соседями.
	type tree struct{ x, y, hash int }
	var trees []tree

	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cell := grid.At(x, y)
			p := domain.Point{X: x, Y: y}
			visible := vis.Visible[p]
			explored := vis.Explored[p]
			hash := cellHash(x, y)

			// Пустота и неисследованное — сплошная чаща: карта выглядит как
			// поляны, прорубленные в лесу, а туман войны — как тёмный лес.
			if cell == domain.SymEmpty || (!visible && !explored) {
				r.drawTile(field, "wall", x, y, camX, camY, hash)
				if !visible {
					r.dimCell(field, x, y, camX, camY)
				} else if r.sprites.Has("tree") && hash%5 == 0 {
					trees = append(trees, tree{x, y, hash})
				}
				continue
			}

			switch {
			case isFloor(cell):
				// Тропы берутся из данных уровня: символ клетки затирается
				// маркерами игрока и предметов, тип земли по нему не узнать.
				role := "floor"
				if paths[p] {
					role = "path"
				}
				r.drawTile(field, role, x, y, camX, camY, hash)

				if cell == domain.SymExit {
					r.drawTile(field, "portal", x, y, camX, camY, 0)
				} else if cell == domain.SymRoomFloor && visible &&
					r.sprites.Has("decor") && hash%11 == 0 {
					r.drawTile(field, "decor", x, y, camX, camY, hash/11)
				}
			case cell == domain.SymWall:
				r.drawTile(field, "wall", x, y, camX, camY, hash)
				if visible && r.sprites.Has("tree") && hash%4 == 0 {
					trees = append(trees, tree{x, y, hash})
				}
			}

			if !visible {
				r.dimCell(field, x, y, camX, camY)
			}
		}
	}

	for _, t := range trees {
		r.drawEntity(field, "tree", t.x, t.y, camX, camY, t.hash, 1)
	}

	for _, it := range s.Level.Items {
		if vis.Visible[domain.Point{X: it.X, Y: it.Y}] {
			r.drawTile(field, itemRole(it.Type), it.X, it.Y, camX, camY, tick)
		}
	}

	for _, op := range s.Level.AllOpponents() {
		if !op.IsAlive() || !op.IsVisible {
			continue
		}
		if !vis.Visible[domain.Point{X: op.X, Y: op.Y}] {
			continue
		}
		r.drawEntity(field, op.Type.SpriteRole(), op.X, op.Y, camX, camY, tick, op.Facing)
	}

	r.drawEntity(field, "player", s.Player.X, s.Player.Y, camX, camY, tick, s.Player.Facing)
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
	img := r.sprites.FrameFacing(role, tick, facing)
	if img == nil {
		return
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(
		float64(x*TileSize+TileSize/2-w/2-camX),
		float64((y+1)*TileSize-h-camY),
	)
	dst.DrawImage(img, op)
}

// dimCell затемняет клетку, которую игрок помнит, но сейчас не видит.
func (r *Renderer) dimCell(dst *ebiten.Image, x, y, camX, camY int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x*TileSize-camX), float64(y*TileSize-camY))
	dst.DrawImage(r.dim, op)
}
