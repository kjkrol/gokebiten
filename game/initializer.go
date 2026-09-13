package game

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// Initializer is what a Stage gets during Init to install plugins and configure the ECS.
type Initializer interface {
	plugin.Installer
	Use(p plugin.Plugin) error

	// World returns the engine's built-in world.Plugin.
	World() *world.Plugin

	// Track saves and loads s alongside the game's Plugins.
	Track(s plugin.Serializable) error

	// TPS returns the engine's measured-ticks-per-second counter.
	TPS() *TPS
}
