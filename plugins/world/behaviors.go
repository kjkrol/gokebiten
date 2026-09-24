package world

import (
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugin/host"
)

// Payload is what world's hosts hand their behaviors: Moving before an entity moves, Leaving while
// it is Outside an open edge, Drawing as it is about to be drawn.
type Payload interface{ Moving | Leaving | Drawing }

// Each is a behavior run on every entity carrying T that the payload's host visits; T must not be
// Base, which every host requires — Every is for those. Register it with Plugin.RegisterBehavior.
func Each[T any, P Payload](react func(t plugin.Tick, state *T, about P)) plugin.Behavior {
	return host.Each(react)
}

// Every is Each without a state component: every entity the payload's host visits.
func Every[P Payload](react func(t plugin.Tick, about P)) plugin.Behavior { return host.Every(react) }
