package game

import (
	"testing"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestBackspaceRemovesWholeRune(t *testing.T) {
	// Обрезка по байту оставила бы в имени половину кириллической буквы,
	// и такая запись уезжала бы в таблице рекордов.
	g := &Game{nameInput: "Юра"}
	g.handleNameEntry(ebiten.KeyBackspace)
	if g.nameInput != "Юр" {
		t.Fatalf("после backspace имя %q, ожидалось \"Юр\"", g.nameInput)
	}
	for i := 0; i < 5; i++ {
		g.handleNameEntry(ebiten.KeyBackspace)
		if !utf8.ValidString(g.nameInput) {
			t.Fatalf("имя стало битым: %q", g.nameInput)
		}
	}
	if g.nameInput != "" {
		t.Fatalf("имя должно стереться полностью, осталось %q", g.nameInput)
	}
}
