package locale

import (
	"strings"
	"testing"
)

func TestTranslationsPreserveProperNames(t *testing.T) {
	for _, name := range []string{"Juggernaut", "Skywrath Mage", "Tango", "Healing Salve", "Phantom Clarity", "Aghanim's Scroll", "Quelling Blade", "YurneROGUE"} {
		if Text(Russian, name) != name || Message(Russian, name) != name {
			t.Fatalf("proper name translated: %s", name)
		}
	}
	cases := map[string]string{
		"PLAY":                          "ИГРАТЬ",
		"Picked up: Tango [+30 HP].":    "Подобрано: Tango [+30 ОЗ].",
		"You hit the Pudge for 12 dmg.": "Удар по Pudge: 12 урона.",
		"The Bloodseeker drained your max HP by 10!":             "Bloodseeker снижает максимальное здоровье на 10!",
		"The Skywrath Mage hit you for 10 dmg. You fall asleep!": "Skywrath Mage наносит 10 урона. Вы засыпаете!",
		"You equipped Yasha. Dropped Quelling Blade.":            "Экипировано: Yasha. Сброшено: Quelling Blade.",
		"You equipped Yasha. Stowed Silver Edge in backpack.":    "Экипировано: Yasha. В рюкзаке: Silver Edge.",
		"Sharpened Desolator.":                                   "Заточено: Desolator.",
		"You used Phantom Clarity [+3 AGI].":                     "Использовано: Phantom Clarity [+3 ЛОВ].",
		"You used Vital Scroll [+5 MAX HP].":                     "Использовано: Vital Scroll [+5 МАКС ОЗ].",
		"You used Moon Clarity [+40 SHIELD].":                    "Использовано: Moon Clarity [+40 ЩИТ].",
		"The Bloodseeker struck your shield.":                    "Bloodseeker бьёт по щиту.",
		"RUN SUMMARY":                                            "ИТОГИ",
		"New game: Starting game...":                             "Новый забег: Запуск игры...",
	}
	for source, want := range cases {
		if got := Message(Russian, source); got != want {
			t.Errorf("%q: got %q, want %q", source, got, want)
		}
		if Message(English, source) != source {
			t.Errorf("English changed: %s", source)
		}
	}
}

func TestRussianTerminologyAndStatSuffixes(t *testing.T) {
	if Text(Russian, "PARRY") != "БЛОК" || Text(English, "PARRY") != "PARRY" {
		t.Fatal("parry marker must read БЛОК in Russian")
	}
	if Text(Russian, "MISS") != "ПРОМАХ" || Text(English, "MISS") != "MISS" {
		t.Fatal("combat miss labels should be localized independently")
	}
	if got := Text(Russian, "Steals your max HP. Deflects your first strike."); got != "Крадёт максимальное здоровье. Блокирует первый удар." {
		t.Fatalf("help should spell out health: %s", got)
	}
	intro := Text(Russian, "Stronger weapons equip, weaker ones sharpen yours. Scrolls apply on pickup. C: food. X: potions.")
	if !strings.Contains(intro, "C: еда") || strings.Contains(intro, "Tango") {
		t.Fatal("intro must describe the food category")
	}
	for key, want := range map[string]string{
		"HELP": "ПОМОЩЬ", "HUD HELP": "ПОМОЩЬ", "FOOD": "ЕДА",
		"CLARITY": "ЗЕЛЬЯ", "CLARITIES": "ЗЕЛЬЯ", "SCROLL": "СВИТКИ", "WEAPON": "ОРУЖИЕ", "PORTAL": "ПОРТАЛ",
	} {
		if Text(Russian, key) != want {
			t.Errorf("wrong category %s", key)
		}
	}
	for source, want := range map[string]string{
		" [+5 STR]": " [+5 СИЛ]", " [+4 AGI]": " [+4 ЛОВ]", " [+10 HP]": " [+10 ОЗ]", " [+5 MAX HP]": " [+5 МАКС ОЗ]", " [+40 SHIELD]": " [+40 ЩИТ]",
		"STR Blade": "STR Blade",
	} {
		if StatSuffix(Russian, source) != want || StatSuffix(English, source) != source {
			t.Errorf("bad stat suffix %s", source)
		}
	}
}

func TestLanguageStorage(t *testing.T) {
	setupStorage(t)
	if Load() != English {
		t.Fatal("first launch must default to English")
	}
	for _, language := range []Language{Russian, English, Russian, "invalid"} {
		Save(language)
		if got := Load(); got != Normalize(language) {
			t.Fatalf("language not saved: %s", got)
		}
	}
}
