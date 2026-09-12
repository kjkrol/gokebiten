// Package gokebiten wires a user-implemented game.Game into Ebitengine's
// Update/Draw/Layout loop. See package game for the interfaces you
// implement, and package plugin for the extension contract.
package gokebiten

import (
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
)

// Engine drives a user-implemented game.Game through the Ebitengine loop.
type Engine = engine.Engine

// Props configures Engine's window and target tick rate.
type Props = engine.Props

// Louch a game
func Run(props *Props, g game.Game) {
	engine.NewEngine(props, g).Run()
}
