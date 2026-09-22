// Package gram wires a user-implemented game.Game into Ebitengine's
// Update/Draw/Layout loop. See package game for the interfaces you
// implement, and package plugin for the extension contract.
package gram

import (
	"github.com/kjkrol/gram/game"
	"github.com/kjkrol/gram/internal/engine"
)

// Run launches a Game
func Run(g game.Game) {
	engine.NewEngine(g).Run()
}
