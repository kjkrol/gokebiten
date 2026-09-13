package game

import "github.com/kjkrol/gokebiten/plugins/world"

// Props configures Engine's window, target tick rate, and the built-in
// world — returned by Game.Props and read once, when the engine starts.
type Props struct {
	Title                     string
	TargetTPS                 int
	ScreenWidth, ScreenHeight int
	World                     world.Config
}
