// Package gokebiten wires a user-implemented game.Game into Ebitengine's
// Update/Draw/Layout loop. See package game for the interfaces you
// implement, and package plugin for the extension contract.
package gokebiten

import (
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
)

// Run launches a Game
func Run(g game.Game) {
	engine.NewEngine(g).Run()
}
