// Package locale translates presentation text without changing simulation data.
package locale

type Language string

const (
	English Language = "en"
	Russian Language = "ru"
)

func Normalize(language Language) Language {
	if language == Russian {
		return Russian
	}
	return English
}

func Text(language Language, key string) string {
	if language == Russian {
		if translated, ok := russian[key]; ok {
			return translated
		}
	}
	return key
}

var russian = map[string]string{
	"PLAY": "ИГРАТЬ", "LEADERBOARD": "РЕКОРДЫ", "HELP": "ПОМОЩЬ",
	"MUSIC": "МУЗЫКА", "SFX": "ЗВУК", "MUSIC %d%%": "МУЗЫКА %d%%", "SFX %d%%": "ЗВУК %d%%",
	"BACK": "НАЗАД", "RESUME": "ПРОДОЛЖИТЬ", "MAIN MENU": "ГЛАВНОЕ МЕНЮ", "CANCEL": "ОТМЕНА",
	"CONTINUE": "ДАЛЕЕ", "PLAY AGAIN": "ЕЩЁ РАЗ", "MENU": "МЕНЮ", "RUN": "БЕГ",
	"SELECT": "ВЫБОР", "USE": "ПРИМ.", "HUD HELP": "ПОМОЩЬ",
	"EVERY STRIKE MAY BE THE LAST": "КАЖДЫЙ УДАР МОЖЕТ СТАТЬ ПОСЛЕДНИМ",
	"YOUR NAME":                    "ВАШЕ ИМЯ", "EMPTY NAME = ANONYMOUS": "БЕЗ ИМЕНИ = АНОНИМ",
	"LEAVE RUN?": "ВЫЙТИ?", "This run will be lost.": "Забег будет потерян",
	"WELCOME": "ДОБРО ПОЖАЛОВАТЬ", "EVERY STEP COUNTS": "КАЖДЫЙ ШАГ ВАЖЕН",
	"REACH THE PORTAL": "НАЙДИТЕ ПОРТАЛ", "MOVE & ATTACK": "ДВИЖЕНИЕ И БОЙ",
	"COLLECT & USE": "ПРЕДМЕТЫ", "TAKE YOUR TIME": "НЕ СПЕШИТЕ",
	"Find the portal on each level. Escape level 21 to win. Gold is your score.":                                                       "Ищите портал на каждом уровне. Пройдите 21 уровень для победы. Золото определяет ваш счёт.",
	"Use WASD or arrows. Step into an enemy to strike with your sword.":                                                                "WASD или стрелки: движение. Шаг на врага: удар мечом. Удерживайте направление для ходьбы.",
	"Use the D-pad. Step into an enemy to strike with your sword.":                                                                     "Крестовина: движение. Шаг на врага: удар мечом. Удерживайте направление для ходьбы.",
	"Pick up loot by walking over it. H/J/K/E: weapons, food, clarities, scrolls.":                                                     "Наступайте на добычу для подбора. H/J/K/E: оружие, еда, зелья, свитки.",
	"Pick up loot by walking over it. Tap an item category, then SELECT to use it.":                                                    "Наступайте на добычу для подбора. Выберите категорию и предмет, затем нажмите кнопку применения.",
	"Enemies act when you take a turn. Food heals, clarity buffs are temporary, scrolls are permanent. MENU pauses; HELP has details.": "Враги ходят после вас. Еда лечит, зелья усиливают временно, свитки навсегда. МЕНЮ: пауза, ПОМОЩЬ: подробности.",
	"ENEMIES": "ВРАГИ", "ITEMS": "ПРЕДМЕТЫ",
	"Tough and slow. Wanders randomly.":               "Крепкий и медленный. Просто бродит.",
	"Steals your max HP. Deflects your first strike.": "Крадёт максимальное здоровье. Блокирует первый удар.",
	"Blinks around the room, mostly invisible.":       "Телепортируется по комнате, почти невидим.",
	"Moves 2 tiles. Rests, counters, never misses.":   "Шагает на 2 клетки. Отдыхает, контратакует, не промахивается.",
	"Moves diagonally. Hits may put you to sleep.":    "Ходит по диагонали. Удар может усыпить.",
	"Restores health.":                                "Восстанавливает здоровье.",
	"Temporary stat buff for 20 turns.":               "Усиление на 20 ходов.",
	"Permanent stat buff.":                            "Усиление навсегда.",
	"Equip it; the old one drops nearby.":             "Возьми его. При замене старое оружие падает рядом.",
	"Descend deeper. Clear level 21 to win.":          "Переход дальше. Пройдите 21 уровень для победы.",
	"YOU DIED":                                        "ВЫ ПОГИБЛИ", "VICTORY": "ПОБЕДА", "RUN SUMMARY / ": "ИТОГИ / ",
	"GOLD": "ЗОЛОТО", "LEVEL": "УРОВЕНЬ", "KILLS": "УБИЙСТВ", "TURNS": "ХОДЫ",
	"ATTACKS": "АТАКИ", "HITS TAKEN": "ПОЛУЧЕНО УДАРОВ", "STEPS": "ШАГИ", "FOOD USED": "СЪЕДЕНО",
	"CLARITIES": "ЗЕЛЬЯ", "CLARITY": "ЗЕЛЬЯ", "SCROLLS": "СВИТКИ", "NAME": "ИМЯ", "LVL": "УР.",
	"WEAPON": "ОРУЖИЕ", "PORTAL": "ПОРТАЛ",
	"ATK": "АТАКИ", "HIT": "УДАРЫ", "FOOD": "ЕДА", "SCROLL": "СВИТКИ",
	"GLOBAL": "ОБЩИЙ РЕЙТИНГ", "NO RECORDS YET": "ПОКА НЕТ РЕКОРДОВ", "LOADING...": "ЗАГРУЗКА...",
	"SERVER UNAVAILABLE": "СЕРВЕР НЕДОСТУПЕН", "CONNECTING": "СОЕДИНЕНИЕ",
	"Starting game...": "Запуск игры...", "Submitting score...": "Отправка результата...",
	"Too many scores. Submission rejected.":                       "Слишком много результатов. Отправка отклонена.",
	"Score rejected by the server.":                               "Сервер отклонил результат.",
	"Score submission could not be confirmed.":                    "Не удалось подтвердить отправку результата.",
	"Score submitted to global leaderboard!":                      "Результат добавлен в общий рейтинг!",
	"Connection timed out. Press PLAY to retry.":                  "Время ожидания истекло. Нажмите ИГРАТЬ для повтора.",
	"Too many starts. Wait, then press PLAY.":                     "Слишком много запусков. Подождите и нажмите ИГРАТЬ.",
	"Server rejected the start. Reload and retry.":                "Сервер отклонил запуск. Обновите страницу.",
	"Cannot reach leaderboard. Press PLAY to retry.":              "Нет связи с сервером. Нажмите ИГРАТЬ для повтора.",
	"Invalid server response. Reload and retry.":                  "Некорректный ответ сервера. Обновите страницу.",
	"Score not submitted: leaderboard unavailable for this game.": "Результат не отправлен: рейтинг недоступен для этого забега.",
	"Game length limit reached: score cannot be submitted.":       "Превышена длина забега: отправка невозможна.",
	"LEVEL %d/%d  GOLD %d":                                        "УРОВЕНЬ %d/%d  ЗОЛОТО %d", "LEVEL %d/%d": "УРОВЕНЬ %d/%d",
	"STRENGTH %d   AGILITY %d": "СИЛА %d   ЛОВКОСТЬ %d", "STRENGTH %d  AGILITY %d": "СИЛА %d  ЛОВКОСТЬ %d",
	"STR": "СИЛ", "AGI": "ЛОВ", "HP": "ОЗ", "MAX HP": "МАКС ОЗ", "MHP": "МОЗ",
	"HEALTH %d / %d": "ЗДОРОВЬЕ %d / %d",
	"SLEEP %dT":      "СОН %dХ", "ZZ %dT": "ZZ %dХ", "%s %+d %dT": "%s %+d %dХ", "MISS": "ПРОМАХ",
	"Can't move that way.": "Туда не пройти.", "You are asleep!": "Вы спите!",
	"Backpack full! Cannot holster weapon.": "Рюкзак полон! Нельзя убрать оружие.",
	"An enemy is in the way.":               "Враг преграждает путь.", "You stop at the portal.": "Вы остановились у портала.",
	"Run: press a direction key.": "Бег: выберите направление.",
	"Player name":                 "Имя игрока", "Tap to enter name": "Нажмите для ввода имени",
	"No other weapons in backpack.": "В рюкзаке нет другого оружия.",
	"No food in backpack.":          "В рюкзаке нет еды.",
	"No clarities in backpack.":     "В рюкзаке нет зелий.",
	"No elixirs in backpack.":       "В рюкзаке нет зелий.",
	"No items in backpack.":         "В рюкзаке нет предметов.",
	"No scrolls in backpack.":       "В рюкзаке нет свитков.",
}
