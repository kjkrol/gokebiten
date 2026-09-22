package gram

import (
	"github.com/kjkrol/gram/game"
	"github.com/kjkrol/gram/internal/engine"
)

// Run launches a Game
func Run(g game.Game) {
	engine.NewEngine(g).Run()
}
