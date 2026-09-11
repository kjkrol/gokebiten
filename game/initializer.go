package game

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// Initializer is what a user's Game gets during Init — everything a
// Plugin can do, plus installing plugins into the engine and reaching Runtime.
type Initializer interface {
	plugin.Installer
	Use(p plugin.Plugin) error

	// World returns the engine's built-in world.Plugin — installed
	// automatically; configure it here (WithRenderer, WithCameraControls)
	// and store it for use in Spawn.
	World() *world.Plugin
}
