package collision

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugin/host"
	"github.com/kjkrol/uid"
)

// Between is a behavior for every confirmed contact where one side carries a and the other b;
// plugin.Any on either side takes whatever is there. Register it with Plugin.RegisterBehavior.
func Between[FA, FB any](a plugin.Tag[FA], b plugin.Tag[FB], react func(t plugin.Tick, m Meeting)) plugin.Behavior {
	return host.Pair(a, b, react)
}

// Each is a behavior run every tick on every Collider carrying T, told what it struck.
func Each[T any](react func(t plugin.Tick, state *T, s Struck)) plugin.Behavior {
	return host.Each(react)
}

// Every is Each without a state component: every Collider, every tick.
func Every(react func(t plugin.Tick, s Struck)) plugin.Behavior { return host.Every(react) }

// Meeting is one confirmed contact as seen from Self: who it met, the impulse exchanged
// (zero when only detected), and the way Self left Other.
type Meeting struct {
	Self, Other uid.UID64
	Impact      float64
	Normal      geom.Vec
}

// Struck is what an Each behavior hosted here is told about an entity:
// which it is, and what it struck the tick before.
type Struck struct {
	ID       uid.UID64
	Contacts []Contact
}
