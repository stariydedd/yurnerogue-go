package render

import (
	"strings"
	"testing"
)

func TestMessageLinesKeepsShortMessageWhole(t *testing.T) {
	got := messageLines("You missed the Pudge.", 40)
	if len(got) != 1 || got[0] != "You missed the Pudge." {
		t.Fatalf("короткое сообщение не должно разбиваться, получено %q", got)
	}
}

func TestMessageLinesSplitsAtColon(t *testing.T) {
	// «Picked up: имя предмета» ломается по двоеточию: заголовок отдельно,
	// длинное имя отдельно — так оно помещается на узком экране.
	msg := "Picked up: Elixir of the Phantom's Breath [+3 STR]."
	got := messageLines(msg, 44)

	if len(got) != 2 {
		t.Fatalf("ожидалось две строки, получено %d: %q", len(got), got)
	}
	if got[0] != "Picked up:" {
		t.Fatalf("первая строка %q, ожидалось «Picked up:»", got[0])
	}
	if !strings.HasPrefix(got[1], "Elixir") {
		t.Fatalf("вторая строка должна начинаться с названия, получено %q", got[1])
	}
	for i, line := range got {
		if len(line) > 44 {
			t.Fatalf("строка %d длиннее лимита: %q", i, line)
		}
	}
}

func TestMessageLinesWrapsWithoutColon(t *testing.T) {
	msg := "The Skywrath Mage hit you for 42 dmg. You fall asleep!"
	got := messageLines(msg, 30)

	if len(got) < 2 {
		t.Fatalf("длинное сообщение должно переноситься, получено %q", got)
	}
	for i, line := range got {
		if len(line) > 30 {
			t.Fatalf("строка %d длиннее лимита: %q", i, line)
		}
	}
	if strings.Join(got, " ") != msg {
		t.Fatalf("перенос потерял или переставил слова: %q", got)
	}
}

func TestWrapTextKeepsAllWords(t *testing.T) {
	s := "Moves 2 tiles. Rests, counters, never misses."
	got := wrapText(s, 20)

	if strings.Join(got, " ") != s {
		t.Fatalf("перенос исказил текст: %q", got)
	}
	for i, line := range got {
		if len(line) > 20 {
			t.Fatalf("строка %d длиннее лимита: %q", i, line)
		}
	}
}

func TestWrapTextHandlesOverlongWord(t *testing.T) {
	// Слово длиннее лимита обрезать нельзя — иначе текст потеряется;
	// оно занимает строку целиком.
	got := wrapText("Parchment-of-Thunderous-Roar tail", 10)
	if len(got) == 0 || got[0] != "Parchment-of-Thunderous-Roar" {
		t.Fatalf("длинное слово должно остаться целым, получено %q", got)
	}
}
