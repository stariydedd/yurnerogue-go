package render

import (
	"testing"
	"unicode/utf8"
)

func sampleRecord(name string) LeaderboardRecord {
	return LeaderboardRecord{
		PlayerName: name, Treasures: 2484, Level: 14, EnemiesKilled: 38,
		FoodUsed: 18, ElixirsUsed: 9, ScrollsRead: 20, AttacksMade: 134,
		HitsTaken: 64, TilesMoved: 2143,
	}
}

func TestLeaderboardRowsAlignAcrossAlphabets(t *testing.T) {
	// Колонки держатся на моноширинном шрифте, поэтому все строки таблицы
	// обязаны состоять из одинакового числа символов — независимо от того,
	// сколько байт занимает имя игрока.
	want := utf8.RuneCountInString(leaderboardHeader(false))
	for _, name := range []string{"stan", "Этофейкн", "темакиби", "", "1"} {
		row := leaderboardRow(6, sampleRecord(name), false)
		if got := utf8.RuneCountInString(row); got != want {
			t.Fatalf("строка с именем %q — %d символов, в шапке %d:\n%s", name, got, want, row)
		}
	}
}

func TestLongNameIsCutByRunesNotBytes(t *testing.T) {
	// Обрезка по байтам разрубила бы кириллический символ пополам.
	name := "лешадскиймальчик" // 16 символов, 32 байта
	row := leaderboardRow(1, sampleRecord(name), false)
	if !utf8.ValidString(row) {
		t.Fatal("в строке появилась битая руна")
	}
	if utf8.RuneCountInString(row) != utf8.RuneCountInString(leaderboardHeader(false)) {
		t.Fatalf("длинное имя сломало ширину строки:\n%s", row)
	}
}

func TestBrokenNameDoesNotBreakRow(t *testing.T) {
	// Старые записи в базе могли остаться с обрезанной по байту руной.
	if got := playerLabel("fle\xd0"); got != "fle" {
		t.Fatalf("битый байт должен вырезаться, получено %q", got)
	}
	// Перевод строки разорвал бы таблицу на две.
	if got := playerLabel("a\nb"); got != "ab" {
		t.Fatalf("управляющие символы должны вырезаться, получено %q", got)
	}
	if got := playerLabel(""); got != "anonymous" {
		t.Fatalf("пустое имя -> %q, ожидалось anonymous", got)
	}
}

func TestNarrowRowsAlignToo(t *testing.T) {
	want := utf8.RuneCountInString(leaderboardHeader(true))
	for _, name := range []string{"stan", "Этофейкн"} {
		if got := utf8.RuneCountInString(leaderboardRow(3, sampleRecord(name), true)); got != want {
			t.Fatalf("узкая строка с именем %q — %d символов, в шапке %d", name, got, want)
		}
	}
}
