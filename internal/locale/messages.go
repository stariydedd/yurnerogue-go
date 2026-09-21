package locale

import (
	"regexp"
	"strings"
)

// Match complete message templates, never replace words inside proper names.
var messages = []struct {
	pattern *regexp.Regexp
	russian string
}{
	{regexp.MustCompile(`^You holstered (.+)\.$`), `Оружие убрано: $1.`},
	{regexp.MustCompile(`^You equipped (.+)\.$`), `Экипировано: $1.`},
	{regexp.MustCompile(`^You used (.+)\.$`), `Использовано: $1.`},
	{regexp.MustCompile(`^Picked up: (.+)\.$`), `Подобрано: $1.`},
	{regexp.MustCompile(`^Backpack full! Cannot pick up (.+)\.$`), `Рюкзак полон! Нельзя подобрать $1.`},
	{regexp.MustCompile(`^You descend to level (\d+)\.$`), `Переход на уровень $1.`},
	{regexp.MustCompile(`^Your first strike against the (.+) was deflected!$`), `$1 блокирует первый удар!`},
	{regexp.MustCompile(`^You missed the (.+)\.$`), `Промах по $1.`},
	{regexp.MustCompile(`^You killed the (.+) for (\d+) dmg! Gained (\d+) gold\.$`), `$1 убит: $2 урона! Получено $3 золота.`},
	{regexp.MustCompile(`^You hit the (.+) for (\d+) dmg\.$`), `Удар по $1: $2 урона.`},
	{regexp.MustCompile(`^The (.+) missed you\.$`), `$1 промахивается.`},
	{regexp.MustCompile(`^The (.+) is preparing to strike\.\.\.$`), `$1 готовится к удару...`},
	{regexp.MustCompile(`^The (.+) drained your max HP by (\d+)!$`), `$1 снижает максимальное здоровье на $2!`},
	{regexp.MustCompile(`^The (.+) hit you for (\d+) dmg\.$`), `$1 наносит $2 урона.`},
}

func Message(language Language, message string) string {
	if language != Russian {
		return message
	}
	if translated := Text(language, message); translated != message {
		return translated
	}
	if tail, ok := strings.CutPrefix(message, "New game: "); ok {
		return "Новый забег: " + Message(language, tail)
	}
	if tail, ok := strings.CutPrefix(message, "RUN SUMMARY / "); ok {
		return Text(language, "RUN SUMMARY / ") + tail
	}
	for _, suffix := range []struct{ en, ru string }{
		{" You fall asleep!", " Вы засыпаете!"},
	} {
		if head, ok := strings.CutSuffix(message, suffix.en); ok {
			return Message(language, head) + suffix.ru
		}
	}
	if head, tail, ok := strings.Cut(message, ". Dropped "); ok {
		return Message(language, head+".") + " Сброшено: " + tail
	}
	if head, tail, ok := strings.Cut(message, ". Stowed "); ok {
		return Message(language, head+".") + " В рюкзаке: " + strings.TrimSuffix(tail, " in backpack.") + "."
	}
	for _, entry := range messages {
		if entry.pattern.MatchString(message) {
			return StatSuffix(language, entry.pattern.ReplaceAllString(message, entry.russian))
		}
	}
	return message
}

var statSuffix = regexp.MustCompile(`\[([+-]\d+) (MAX HP|STR|AGI|HP)\]`)

// Translate only stat suffixes, not words in item or player names.
func StatSuffix(language Language, label string) string {
	if language != Russian {
		return label
	}
	return statSuffix.ReplaceAllStringFunc(label, func(suffix string) string {
		parts := statSuffix.FindStringSubmatch(suffix)
		return "[" + parts[1] + " " + Text(language, parts[2]) + "]"
	})
}
