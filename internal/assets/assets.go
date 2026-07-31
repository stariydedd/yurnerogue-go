// Package assets зашивает графику и шрифт в бинарник, чтобы браузеру не
// пришлось делать отдельные запросы за каждым файлом.
package assets

import "embed"

// FS содержит custom/ (пиксель-арт по ролям) и fonts/ (Press Start 2P).
// Лицензии и авторы перечислены в LICENSE.txt рядом.
//
//go:embed custom fonts
var FS embed.FS
