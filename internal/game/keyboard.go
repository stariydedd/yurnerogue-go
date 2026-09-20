package game

import "github.com/hajimehoshi/ebiten/v2"

// Only the most recently pressed direction repeats; keys held across a menu
// or focus change must be released and pressed again before moving.
type keyboardInput struct {
	held []ebiten.Key
	next int
}

func (k *keyboardInput) reset() {
	k.held = k.held[:0]
	k.next = 0
}

func (k *keyboardInput) update(pressed []ebiten.Key, down func(ebiten.Key) bool) ebiten.Key {
	previous := ebiten.KeyMax
	if len(k.held) > 0 {
		previous = k.held[len(k.held)-1]
	}
	kept := k.held[:0]
	for _, key := range k.held {
		if down(key) {
			kept = append(kept, key)
		}
	}
	k.held = kept
	newDirection := false
	for _, key := range pressed {
		if _, ok := directionKeys[key]; !ok {
			continue
		}
		k.held = append(k.held, key)
		newDirection = true
	}
	if len(k.held) == 0 {
		k.next = 0
		return ebiten.KeyMax
	}
	key := k.held[len(k.held)-1]
	if newDirection {
		k.next = repeatDelay
		return key
	}
	if key != previous {
		k.next = repeatInterval
		return ebiten.KeyMax
	}
	k.next--
	if k.next <= 0 {
		k.next = repeatInterval
		return key
	}
	return ebiten.KeyMax
}

func (g *Game) handleKeyboard(pressed []ebiten.Key, down func(ebiten.Key) bool, focused bool) {
	if !focused {
		g.keyboard.reset()
		return
	}
	state := g.state
	if state != StatePlaying {
		g.keyboard.reset()
		for _, key := range pressed {
			g.HandleKey(key)
			if g.state != state {
				break
			}
		}
		return
	}
	for _, key := range pressed {
		if _, direction := directionKeys[key]; !direction {
			g.HandleKey(key)
			if g.state != state {
				g.keyboard.reset()
				return
			}
		}
	}
	if key := g.keyboard.update(pressed, down); key != ebiten.KeyMax {
		g.HandleKey(key)
	}
}
