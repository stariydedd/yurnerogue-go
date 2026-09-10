// terrainpreview captures the real renderer using a local fixture, without
// network requests or score submissions.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
	"github.com/stariydedd/yurnerogue-go/internal/render"
)

type preview struct {
	r         *render.Renderer
	s         *domain.Session
	out       string
	done      bool
	err       error
	frames    int
	firstHash [32]byte
	walk      []domain.Point
	step      int
	page      string
}

func (p *preview) Update() error {
	if p.err != nil {
		return p.err
	}
	if p.done {
		return ebiten.Termination
	}
	return nil
}

func (p *preview) Draw(screen *ebiten.Image) {
	if p.done {
		return
	}
	switch p.page {
	case "menu":
		p.r.DrawMainMenu(screen, 0, "")
		p.r.DrawMenuButtons(screen, render.MenuHome, 0)
	case "help":
		p.r.DrawHelp(screen)
		p.r.DrawMenuButtons(screen, render.MenuBack, 0)
	case "name":
		p.r.DrawNameEntry(screen, "")
		p.r.DrawMenuButtons(screen, render.MenuName, 0)
	default:
		p.r.DrawWorld(screen, p.s)
		p.r.DrawHUD(screen, p.s)
		if p.r.Layout.Touch {
			render.NewControls(p.r.Layout).Draw(screen, p.r, nil, render.SelectHelp, true, p.s.Player)
		}
	}
	pixels := make([]byte, 4*screen.Bounds().Dx()*screen.Bounds().Dy())
	screen.ReadPixels(pixels)
	hash := sha256.Sum256(pixels)
	if p.frames > 0 && hash != p.firstHash {
		p.err = fmt.Errorf("cached terrain frame differs from initial frame")
		return
	}
	p.firstHash = hash
	p.frames++
	if p.frames < 3 {
		return
	}
	destination := p.out
	if len(p.walk) > 0 {
		ext := filepath.Ext(destination)
		destination = fmt.Sprintf("%s-step-%d%s", destination[:len(destination)-len(ext)], p.step, ext)
	}
	f, err := os.Create(destination)
	if err == nil {
		err = png.Encode(f, screen)
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}
	p.err = err
	if err == nil && p.step < len(p.walk) {
		next := p.walk[p.step]
		p.s.Player.X, p.s.Player.Y = next.X, next.Y
		p.step++
		p.frames = 0
		return
	}
	p.done = true
}

func (p *preview) Layout(_, _ int) (int, int) {
	return p.r.Layout.ScreenW, p.r.Layout.ScreenH
}

func fixture(scene string) *domain.Session {
	s := domain.NewSessionSeed(21)
	if scene == "seed" {
		return s
	}
	if scene == "corridor-bend" {
		s.Level.Rooms = nil
		s.Level.Items = nil
		s.Level.Passages = []domain.Rect{{X: 8, Y: 19, W: 13, H: 3}, {X: 18, Y: 12, W: 3, H: 10}}
		s.Level.Doors = nil
		s.Level.Exit = domain.Point{X: 40, Y: 40}
		s.Player.X, s.Player.Y = 19, 20
		s.VisitedRooms = map[int]bool{}
		return s
	}
	room := &domain.Room{X: 8, Y: 6, W: 16, H: 9}
	s.Level.Rooms = []*domain.Room{room, {X: 30, Y: 8, W: 7, H: 6}}
	s.Level.Passages = []domain.Rect{{X: 23, Y: 9, W: 8, H: 3}}
	s.Level.Doors = domain.DoorCells(s.Level.Rooms, s.Level.Passages)
	s.Level.Exit = domain.Point{X: 21, Y: 8}
	s.Player.X, s.Player.Y = 16, 10
	s.Level.Items = []*domain.Item{
		{Type: domain.ItemWeapon, X: 17, Y: 10},
		{Type: domain.ItemFood, X: 12, Y: 8},
		{Type: domain.ItemElixir, X: 19, Y: 13},
		{Type: domain.ItemScroll, X: 11, Y: 12},
		{Type: domain.ItemWeapon, X: 27, Y: 10},
	}
	for i, kind := range []domain.OpponentType{domain.Zombie, domain.Ogre} {
		op := domain.NewOpponent(kind)
		op.X, op.Y = 20+i*2, 11
		room.Enemies = append(room.Enemies, op)
	}
	if scene == "corridor" {
		s.Player.X, s.Player.Y = 26, 10
		s.VisitedRooms[0] = true
	}
	if scene == "heroes" {
		room.Enemies = nil
		positions := []domain.Point{{X: 13, Y: 9}, {X: 16, Y: 8}, {X: 19, Y: 9}, {X: 14, Y: 12}, {X: 18, Y: 12}}
		for i, kind := range domain.AllOpponentTypes {
			op := domain.NewOpponent(kind)
			op.X, op.Y, op.IsVisible = positions[i].X, positions[i].Y, true
			room.Enemies = append(room.Enemies, op)
		}
	}
	if scene == "hero-depth" || scene == "hero-depth-reverse" {
		room.Enemies = nil
		s.Level.Items = nil
		for _, kind := range domain.AllOpponentTypes {
			if kind.SpriteRole() != "skywrath" {
				continue
			}
			op := domain.NewOpponent(kind)
			op.X, op.Y, op.IsVisible = s.Player.X, s.Player.Y+1, true
			if scene == "hero-depth-reverse" {
				op.Y = s.Player.Y - 1
			}
			room.Enemies = append(room.Enemies, op)
		}
	}
	if scene == "gate-items" || scene == "gate-clear" || scene == "gate-hero" {
		room.Enemies = nil
		s.Level.Exit = domain.Point{X: 18, Y: 11}
		s.Player.X, s.Player.Y = 15, 11
		s.Level.Items = nil
		for i, offset := range []domain.Point{{X: 0, Y: -1}, {X: 1, Y: -1}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}, {X: -1, Y: 1}, {X: -1, Y: 0}, {X: -1, Y: -1}} {
			kind := []domain.ItemType{domain.ItemElixir, domain.ItemFood, domain.ItemScroll, domain.ItemWeapon}[i%4]
			s.Level.Items = append(s.Level.Items, &domain.Item{Type: kind, X: s.Level.Exit.X + offset.X, Y: s.Level.Exit.Y + offset.Y})
		}
		if scene == "gate-clear" || scene == "gate-hero" {
			s.Level.Items = nil
		}
		if scene == "gate-hero" {
			s.Player.X, s.Player.Y = 18, 10
		}
	}
	return s
}

func main() {
	scene := flag.String("scene", "room", "room, corridor, corridor-bend (four walking frames), heroes, hero-depth, hero-depth-reverse, gate-items, gate-clear, gate-hero, or seeded gameplay (seed)")
	out := flag.String("out", "build/radiant-room.png", "screenshot destination")
	touch := flag.Bool("touch", false, "capture the portrait touch layout")
	width := flag.Int("width", 480, "touch viewport width")
	height := flag.Int("height", 960, "touch viewport height")
	hud := flag.Bool("hud", false, "populate inventory and temporary effects for a HUD preview")
	frame := flag.Int("frame", 0, "fixed animation frame for all sprite roles")
	page := flag.String("page", "game", "screen to capture: game, menu, help, name")
	left := flag.Bool("left", false, "face all characters left")
	flag.Parse()
	if *page != "game" && *page != "menu" && *page != "help" && *page != "name" {
		log.Fatal("unknown preview page")
	}
	if *scene != "room" && *scene != "corridor" && *scene != "corridor-bend" && *scene != "seed" && *scene != "heroes" && *scene != "hero-depth" && *scene != "hero-depth-reverse" && *scene != "gate-items" && *scene != "gate-clear" && *scene != "gate-hero" {
		log.Fatal("unknown scene")
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		log.Fatal(err)
	}
	l := render.DesktopLayout()
	if *touch {
		l = render.TouchLayout(*width, *height)
	}
	r, err := render.New(l)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(l.ScreenW, l.ScreenH)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetRunnableOnUnfocused(true)
	if *frame < 0 || *frame > 1000 {
		log.Fatal("frame must be between 0 and 1000")
	}
	for i := 0; i < *frame*render.AnimFrameTicks; i++ {
		r.Tick()
	}
	s := fixture(*scene)
	if *hud {
		s.Player.Weapon = &domain.Item{Type: domain.ItemWeapon, Name: "Yasha", StrengthEffect: 35}
		s.Player.Treasures = 120
		for i, kind := range []domain.ItemType{domain.ItemFood, domain.ItemElixir, domain.ItemScroll} {
			for n := 0; n < 3-i; n++ {
				s.Player.PickUpItem(&domain.Item{Type: kind})
			}
		}
		elixir := &domain.Item{Type: domain.ItemElixir, AgilityEffect: 5, StrengthEffect: 3, MaxHealthEffect: 20}
		s.Player.PickUpItem(elixir)
		s.Player.UseItem(elixir)
		s.Player.TakeDamage(80)
		s.Message = "Picked up: Tango."
	}
	if *left {
		s.Player.Facing = -1
		for _, op := range s.Level.AllOpponents() {
			op.Facing = -1
		}
	}
	p := &preview{r: r, s: s, out: *out, page: *page}
	if *scene == "corridor-bend" {
		p.walk = []domain.Point{{X: 19, Y: 19}, {X: 19, Y: 20}, {X: 19, Y: 19}}
	}
	if err := ebiten.RunGame(p); err != nil {
		log.Fatal(err)
	}
}
