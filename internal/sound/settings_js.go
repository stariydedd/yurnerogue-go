//go:build js

package sound

import (
	"encoding/json"
	"syscall/js"
)

func loadSettings() (s Settings) {
	s = Defaults()
	defer func() {
		if recover() != nil {
			s = Defaults()
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", "yurnerogue.audio")
	if v.Type() != js.TypeString || json.Unmarshal([]byte(v.String()), &s) != nil || !s.Valid() {
		return Defaults()
	}
	return s
}
func saveSettings(s Settings) {
	defer func() { _ = recover() }()
	data, _ := json.Marshal(s)
	js.Global().Get("localStorage").Call("setItem", "yurnerogue.audio", string(data))
}
