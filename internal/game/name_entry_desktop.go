//go:build !js

package game

func (g *Game) syncBrowserNameEntry() bool { return false }
