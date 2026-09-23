package selection

import "github.com/kjkrol/gram/plugin"

// Family is selection's tag family: Selectable and Selected live in it.
type Family struct{}

// Tags is selection's tags: Selectable marks an entity the player may select, Selected one
// the player has. A kind gives Selectable with kind.Tagged; the plugin flips Selected.
type Tags struct {
	Selectable, Selected plugin.Tag[Family]
}
