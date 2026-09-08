package render

import "testing"

func TestLoadSpritesAppliesRadiantTerrain(t *testing.T) {
	s, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"floor", "decor", "tree", "bush", "ruins", "verge", "moss", "meadow", "trail"} {
		if len(s.frames[role]) != 4 {
			t.Fatalf("%s: Radiant must override the base sheet with four frames", role)
		}
		for _, frame := range s.frames[role] {
			w, h := TileSize, TileSize
			switch role {
			case "tree":
				w, h = 96, 128
			case "meadow", "trail":
				w, h = 128, 128
			case "bush":
				w, h = 48, 40
			case "ruins":
				w, h = 64, 80
			case "verge":
				w, h = 64, 32
			case "moss":
				w, h = 128, 64
			}
			if frame.Bounds().Dx() != w || frame.Bounds().Dy() != h {
				t.Fatalf("%s: unexpected frame bounds: %v", role, frame.Bounds())
			}
		}
	}
	for _, role := range []string{"player", "pudge"} {
		if !s.Has(role) {
			t.Fatalf("missing unchanged base sprite %q", role)
		}
	}
}

func TestParseSpriteName(t *testing.T) {
	cases := []struct {
		file  string
		role  string
		count int
	}{
		{"food.png", "food", 1},
		{"player.8.png", "player", 8},
		{"riki.39.png", "riki", 39},
		{"ui_run.png", "ui_run", 1},
		// Число обязано быть положительным, иначе это часть имени.
		{"weird.0.png", "weird.0", 1},
		{"weird.x.png", "weird.x", 1},
	}
	for _, c := range cases {
		role, count := parseSpriteName(c.file)
		if role != c.role || count != c.count {
			t.Fatalf("%s -> (%q, %d), ожидалось (%q, %d)", c.file, role, count, c.role, c.count)
		}
	}
}

func TestTileRolesCoverMapTiles(t *testing.T) {
	// Тайлы масштабируются ровно в клетку; если роль выпадет из списка,
	// в сетке появятся щели.
	for _, role := range []string{"floor"} {
		if !tileRoles[role] {
			t.Fatalf("роль %q должна масштабироваться в клетку", role)
		}
	}
	// Персонажи и предметы наоборот: их спрайты выше клетки.
	for _, role := range []string{"player", "pudge", "sword", "heart", "portal"} {
		if tileRoles[role] {
			t.Fatalf("роль %q не должна ужиматься в клетку", role)
		}
	}
}

func TestLoadSpritesExcludesLegacyTerrainRoles(t *testing.T) {
	s, err := LoadSprites()
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"path", "wall", "canopy"} {
		if s.Has(role) {
			t.Fatalf("unused terrain role loaded: %s", role)
		}
	}
}
