//go:build js

package sound

import (
	"syscall/js"
	"testing"
)

func setupTestSettingsStorage(t *testing.T) {
	t.Helper()
	global := js.Global()
	previous := global.Get("localStorage")
	values := make(map[string]string)
	getItem := js.FuncOf(func(_ js.Value, args []js.Value) any {
		value, ok := values[args[0].String()]
		if !ok {
			return nil
		}
		return value
	})
	setItem := js.FuncOf(func(_ js.Value, args []js.Value) any {
		values[args[0].String()] = args[1].String()
		return nil
	})
	storage := global.Get("Object").New()
	storage.Set("getItem", getItem)
	storage.Set("setItem", setItem)
	global.Set("localStorage", storage)
	t.Cleanup(func() {
		if previous.IsUndefined() {
			global.Delete("localStorage")
		} else {
			global.Set("localStorage", previous)
		}
		getItem.Release()
		setItem.Release()
	})
}
