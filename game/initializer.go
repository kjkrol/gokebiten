package game

import "github.com/kjkrol/gokebiten/plugin"

// Initializer is what a user's Game gets during Init — everything a
// Plugin can do, plus installing plugins into the engine and reaching Runtime.
type Initializer interface {
	plugin.Installer
	Use(p plugin.Plugin) error
	Runtime() Runtime
}
