//go:build js

package leaderboard

import (
	"errors"
	"syscall/js"
)

// New создаёт клиент для браузера: запросы идут на тот же origin, откуда
// пришла страница, поэтому базовый адрес не нужен.
func New() *Client { return &Client{} }

// do выполняет запрос через родной fetch браузера.
//
// Вызывается из горутины: ожидание на канале уступает управление планировщику
// Go, тот отдаёт его циклу событий JS, и картинка не замирает.
func (c *Client) do(method, path string, body []byte) ([]byte, error) {
	opts := map[string]any{"method": method}
	if body != nil {
		opts["body"] = string(body)
		opts["headers"] = map[string]any{"Content-Type": "application/json"}
	}

	resp, err := await(js.Global().Call("fetch", c.BaseURL+path, opts))
	if err != nil {
		return nil, err
	}
	if !resp.Get("ok").Bool() {
		return nil, ErrUnavailable
	}

	text, err := await(resp.Call("text"))
	if err != nil {
		return nil, err
	}
	return []byte(text.String()), nil
}

// await дожидается JS-промиса и возвращает его результат.
func await(promise js.Value) (js.Value, error) {
	type outcome struct {
		value js.Value
		err   error
	}
	// Буфер на единицу: промис вызывает ровно один обработчик, но буфер
	// страхует от утечки горутины, если бы сработали оба.
	ch := make(chan outcome, 1)

	then := js.FuncOf(func(_ js.Value, args []js.Value) any {
		var v js.Value
		if len(args) > 0 {
			v = args[0]
		}
		ch <- outcome{value: v}
		return nil
	})
	defer then.Release()

	catch := js.FuncOf(func(_ js.Value, args []js.Value) any {
		msg := "fetch failed"
		if len(args) > 0 {
			if m := args[0].Get("message"); m.Type() == js.TypeString {
				msg = m.String()
			}
		}
		ch <- outcome{err: errors.New(msg)}
		return nil
	})
	defer catch.Release()

	promise.Call("then", then).Call("catch", catch)

	res := <-ch
	return res.value, res.err
}
