package render

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"path"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/stariydedd/yurnerogue-go/internal/assets"
)

// tileRoles масштабируются ровно в клетку, чтобы в сетке не было щелей.
var tileRoles = map[string]bool{"floor": true}

// Sprites — кадры по ролям. Имя файла задаёт роль: `<роль>.png` — статичный
// спрайт, `<роль>.N.png` — N кадров анимации по горизонтали.
type Sprites struct {
	frames        map[string][]*ebiten.Image
	flipped       map[string][]*ebiten.Image
	groundPickups map[string][]*ebiten.Image
	worldHeroes   map[string][]*ebiten.Image
}

// LoadSprites читает базовые PNG, затем применяет оформление Radiant.
func LoadSprites() (*Sprites, error) {
	s := &Sprites{
		frames:  map[string][]*ebiten.Image{},
		flipped: map[string][]*ebiten.Image{},
	}
	for _, dir := range []string{"custom", "custom/radiant"} {
		if err := s.loadDir(dir); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Sprites) loadDir(dir string) error {
	entries, err := assets.FS.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		role, count := parseSpriteName(e.Name())
		file := path.Join(dir, e.Name())
		data, err := assets.FS.ReadFile(file)
		if err != nil {
			return err
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		if img.Bounds().Dx()%count != 0 || img.Bounds().Dx()/count == 0 {
			return fmt.Errorf("%s: width must be a positive multiple of %d frames", file, count)
		}
		s.frames[role] = splitFrames(ebiten.NewImageFromImage(img), count, tileRoles[role])
		if isPickupRole(role) || isHeroRole(role) {
			frames := make([]*ebiten.Image, count)
			w := img.Bounds().Dx() / count
			for i := range frames {
				b := image.Rect(img.Bounds().Min.X+i*w, img.Bounds().Min.Y, img.Bounds().Min.X+(i+1)*w, img.Bounds().Max.Y)
				frames[i] = ebiten.NewImageFromImage(outlineSprite(img, b))
			}
			if isPickupRole(role) {
				if s.groundPickups == nil {
					s.groundPickups = map[string][]*ebiten.Image{}
				}
				s.groundPickups[role] = frames
			} else {
				if s.worldHeroes == nil {
					s.worldHeroes = map[string][]*ebiten.Image{}
				}
				s.worldHeroes[role] = frames
			}
		}
		delete(s.flipped, role)
	}
	return nil
}

// parseSpriteName разбирает имя файла в роль и число кадров:
// `player.8.png` -> ("player", 8), `food.png` -> ("food", 1).
func parseSpriteName(name string) (string, int) {
	stem := strings.TrimSuffix(name, ".png")
	if i := strings.LastIndex(stem, "."); i > 0 {
		if n, err := strconv.Atoi(stem[i+1:]); err == nil && n > 0 {
			return stem[:i], n
		}
	}
	return stem, 1
}

// splitFrames режет горизонтальную полоску на кадры; тайлы дополнительно
// подгоняются под размер клетки.
func splitFrames(sheet *ebiten.Image, count int, isTile bool) []*ebiten.Image {
	w := sheet.Bounds().Dx() / count
	h := sheet.Bounds().Dy()
	frames := make([]*ebiten.Image, 0, count)
	for i := 0; i < count; i++ {
		frame := sheet.SubImage(image.Rect(i*w, 0, (i+1)*w, h)).(*ebiten.Image)
		if isTile && (w != TileSize || h != TileSize) {
			scaled := ebiten.NewImage(TileSize, TileSize)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(TileSize)/float64(w), float64(TileSize)/float64(h))
			scaled.DrawImage(frame, op)
			frame = scaled
		}
		frames = append(frames, frame)
	}
	return frames
}

// Has сообщает, есть ли спрайт для роли.
func (s *Sprites) Has(role string) bool { return len(s.frames[role]) > 0 }

// Frame возвращает кадр роли; tick перебирает кадры по кругу.
func (s *Sprites) Frame(role string, tick int) *ebiten.Image {
	frames := s.frames[role]
	if len(frames) == 0 {
		return nil
	}
	return frames[((tick%len(frames))+len(frames))%len(frames)]
}

// FrameFlipped — кадр, отражённый по горизонтали (персонаж смотрит влево).
// Зеркальные кадры считаются один раз и кэшируются.
func (s *Sprites) FrameFlipped(role string, tick int) *ebiten.Image {
	frames := s.frames[role]
	if len(frames) == 0 {
		return nil
	}
	mirrored, ok := s.flipped[role]
	if !ok {
		mirrored = make([]*ebiten.Image, len(frames))
		for i, f := range frames {
			w, h := f.Bounds().Dx(), f.Bounds().Dy()
			dst := ebiten.NewImage(w, h)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(-1, 1)
			op.GeoM.Translate(float64(w), 0)
			dst.DrawImage(f, op)
			mirrored[i] = dst
		}
		s.flipped[role] = mirrored
	}
	return mirrored[((tick%len(mirrored))+len(mirrored))%len(mirrored)]
}

// FrameFacing выбирает кадр с учётом направления взгляда.
func (s *Sprites) FrameFacing(role string, tick, facing int) *ebiten.Image {
	if facing < 0 {
		return s.FrameFlipped(role, tick)
	}
	return s.Frame(role, tick)
}
