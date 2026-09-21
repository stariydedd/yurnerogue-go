//go:build js

package locale

import "syscall/js"

func Load() (language Language) {
	language = English
	defer func() { _ = recover() }()
	value := js.Global().Get("localStorage").Call("getItem", "yurnerogue.language")
	if value.Type() == js.TypeString {
		language = Normalize(Language(value.String()))
	}
	return
}

func Save(language Language) {
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", "yurnerogue.language", string(Normalize(language)))
}
