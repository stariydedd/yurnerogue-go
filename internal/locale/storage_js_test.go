//go:build js

package locale

import (
	"syscall/js"
	"testing"
)

func setupStorage(t *testing.T) {
	t.Helper()
	global := js.Global()
	previous := global.Get("localStorage")
	values := map[string]string{}
	get := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if value, ok := values[args[0].String()]; ok {
			return value
		}
		return nil
	})
	set := js.FuncOf(func(_ js.Value, args []js.Value) any {
		values[args[0].String()] = args[1].String()
		return nil
	})
	storage := global.Get("Object").New()
	storage.Set("getItem", get)
	storage.Set("setItem", set)
	global.Set("localStorage", storage)
	t.Cleanup(func() {
		global.Set("localStorage", previous)
		get.Release()
		set.Release()
	})
}

func TestUnavailableBrowserStorageDoesNotPreventSwitching(t *testing.T) {
	global := js.Global()
	previous := global.Get("localStorage")
	global.Set("localStorage", js.Undefined())
	t.Cleanup(func() { global.Set("localStorage", previous) })
	Save(Russian)
	if Load() != English || Text(Russian, "PLAY") != "ИГРАТЬ" {
		t.Fatal("missing storage should only prevent persistence")
	}
}
