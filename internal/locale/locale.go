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
	"ABILITIES": "СПОСОБНОСТИ", "CRITICAL STRIKE": "КРИТИЧЕСКИЙ УДАР", "PARRY ABILITY": "ПАРИРОВАНИЕ",
	"You wait.":                             "Вы ждёте.",
	"Wait a turn: let enemies come to you.": "Пропустить ход: пусть враги подойдут сами.",
	"D-pad centre":                          "Центр крестовины",
	"CONTROLS":                              "УПРАВЛЕНИЕ", "GLOSSARY": "ГЛОССАРИЙ", "KEYS": "КЛАВИШИ", "INTERFACE": "ИНТЕРФЕЙС", "BUTTONS": "КНОПКИ",
	"Then a direction: damage x1.5. Recharges after 3 attacks.":                                              "Затем направление: урон x1,5. Заряжается за 3 атаки.",
	"Next to an enemy: blocks hits for a turn and strikes back. Recharges after 3 hits taken.":               "Рядом с врагом: блок ударов на ход и ответный удар. Заряжается за 3 полученных удара.",
	"Tap, then a direction: damage x1.5. Recharges after 3 attacks.":                                         "Нажмите и выберите направление: урон x1,5. Заряжается за 3 атаки.",
	"Tap next to an enemy: blocks hits for a turn and strikes attackers back. Recharges after 3 hits taken.": "Нажмите рядом с врагом: блокирует удары на ход и бьёт атакующих в ответ. Заряжается за 3 полученных удара.",
	"Move": "Ходьба", "Run": "Бег", "Wait": "Ожидание", "Pause": "Пауза", "Help": "Справка", "Fullscreen": "Полный экран", "D-pad": "Крестовина",
	"Food / clarity": "Еда / зелья",
	"Move. Step into an enemy to attack. Hold to keep walking.":            "Движение. Шаг на врага: атака. Удерживайте для ходьбы.",
	"Step into an enemy to attack. Hold to keep walking. Arrows work too.": "Шаг на врага: атака. Удерживайте для ходьбы. Стрелки тоже работают.",
	"Then a direction: run until something blocks the way.":                "Затем направление: бег, пока что-нибудь не преградит путь.",
	"Eat one: restores health.":                                            "Съесть: восстанавливает здоровье.", "Drink one: stat buff for 20 turns.": "Выпить: усиление на 20 ходов.",
	"Choose an item, then USE.":    "Выберите предмет и нажмите ПРИМ.",
	"Pause: resume, volume, exit.": "Пауза: продолжить, громкость, выход.",
	"Open this help.":              "Открыть эту справку.",
	"Resume, volume, exit.":        "Продолжить, громкость, выход.", "Window or full screen.": "Окно или полный экран.",
	"Stronger weapons equip, weaker ones sharpen yours. Scrolls apply on pickup. C: food. X: potions.":                                  "Сильное оружие заменяет текущее, слабое затачивает его. Свитки применяются при подборе. C: еда. X: зелья.",
	"Stronger weapons equip, weaker ones sharpen yours. Scrolls apply on pickup. Select food or potions to use.":                        "Сильное оружие заменяет текущее, слабое затачивает его. Свитки применяются при подборе. Для применения выберите еду или зелье.",
	"F, then direction: critical strike x1.5, recharges after 3 attacks. E next to an enemy: parry the hit and strike back.":            "F и направление: критический удар x1,5, заряжается за 3 атаки. E рядом с врагом: парирование удара и ответный удар.",
	"Critical Strike, then direction: damage x1.5, recharges after 3 attacks. Parry next to an enemy: blocks the hit and strikes back.": "Критический удар и направление: урон x1,5, заряжается за 3 атаки. Парирование рядом с врагом: блок удара и ответный удар.",
	"Permanent buff applied on pickup.":        "Постоянное усиление при подборе.",
	"Equips if stronger, else sharpens yours.": "Заменяет текущее, если сильнее, иначе затачивает его.",
	"Critical Strike: choose a direction.":     "Критический удар: выберите направление.",
	"No enemy next to you.":                    "Рядом нет врага.",
	"Parry: hits are blocked and struck back.": "Парирование: удары блокируются, атакующие получают ответный удар.",
	"No enemy in that direction.":              "В этом направлении нет врага.",
	"%dT":                                      "%dХ",
	"PLAY":                                     "ИГРАТЬ", "LEADERBOARD": "РЕКОРДЫ", "HELP": "ПОМОЩЬ",
	"MUSIC": "МУЗЫКА", "SFX": "ЗВУК", "MUSIC %d%%": "МУЗЫКА %d%%", "SFX %d%%": "ЗВУК %d%%",
	"BACK": "НАЗАД", "RESUME": "ПРОДОЛЖИТЬ", "MAIN MENU": "ГЛАВНОЕ МЕНЮ", "CANCEL": "ОТМЕНА",
	"CONTINUE": "ДАЛЕЕ", "PLAY AGAIN": "ЕЩЁ РАЗ", "MENU": "МЕНЮ", "RUN": "БЕГ",
	"SELECT": "ВЫБОР", "USE": "ПРИМ.", "HUD HELP": "ПОМОЩЬ",
	"EVERY STRIKE MAY BE THE LAST": "КАЖДЫЙ УДАР МОЖЕТ СТАТЬ ПОСЛЕДНИМ",
	"YOUR NAME":                    "ВАШЕ ИМЯ", "EMPTY NAME = ANONYMOUS": "БЕЗ ИМЕНИ = АНОНИМ",
	"LEAVE RUN?": "ВЫЙТИ?", "This run will be lost.": "Забег будет потерян",
	"WELCOME": "ДОБРО ПОЖАЛОВАТЬ", "EVERY STEP COUNTS": "КАЖДЫЙ ШАГ ВАЖЕН",
	"REACH THE PORTAL": "НАЙДИТЕ ПОРТАЛ", "MOVE & ATTACK": "ДВИЖЕНИЕ И БОЙ",
	"COLLECT & USE": "ПРЕДМЕТЫ",
	"Find the portal on each level. Escape level 21 to win. Gold is your score.": "Ищите портал на каждом уровне. Пройдите 21 уровень для победы. Золото определяет ваш счёт.",
	"Use WASD or arrows. Step into an enemy to strike with your sword.":          "WASD или стрелки: движение. Шаг на врага: удар мечом. Удерживайте направление для ходьбы.",
	"Use the D-pad. Step into an enemy to strike with your sword.":               "Крестовина: движение. Шаг на врага: удар мечом. Удерживайте направление для ходьбы.",
	"ENEMIES": "ВРАГИ", "ITEMS": "ПРЕДМЕТЫ",
	"Tough and slow. Wanders randomly.":               "Крепкий и медленный. Просто бродит.",
	"Steals your max HP. Deflects your first strike.": "Крадёт максимальное здоровье. Блокирует первый удар.",
	"Blinks around the room, mostly invisible.":       "Телепортируется по комнате, почти невидим.",
	"Moves 2 tiles. Rests, counters, never misses.":   "Шагает на 2 клетки. Отдыхает, контратакует, не промахивается.",
	"Moves diagonally. Hits may put you to sleep.":    "Ходит по диагонали. Удар может усыпить.",
	"Restores health.":                                "Восстанавливает здоровье.",
	"Temporary stat buff for 20 turns.":               "Усиление на 20 ходов.",
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
	"SLEEP":          "СОН",
	"MISS":           "ПРОМАХ", "PARRY": "БЛОК",
	"Can't move that way.": "Туда не пройти.", "You are asleep!": "Вы спите!",
	"Backpack full! Cannot holster weapon.": "Рюкзак полон! Нельзя убрать оружие.",
	"An enemy is in the way.":               "Враг преграждает путь.", "You stop at the portal.": "Вы остановились у портала.",
	"Run: press a direction key.": "Бег: выберите направление.",
	"Player name":                 "Имя игрока", "Tap to enter name": "Нажмите для ввода имени",
	"No food in backpack.":      "В рюкзаке нет еды.",
	"No clarities in backpack.": "В рюкзаке нет зелий.",
	"No elixirs in backpack.":   "В рюкзаке нет зелий.",
	"No items in backpack.":     "В рюкзаке нет предметов.",
}
