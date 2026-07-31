package render

import "testing"

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
	for _, role := range []string{"floor", "path", "wall", "portal"} {
		if !tileRoles[role] {
			t.Fatalf("роль %q должна масштабироваться в клетку", role)
		}
	}
	// Персонажи и предметы наоборот: их спрайты выше клетки.
	for _, role := range []string{"player", "pudge", "sword", "heart"} {
		if tileRoles[role] {
			t.Fatalf("роль %q не должна ужиматься в клетку", role)
		}
	}
}
