package collision

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/uid"
)

// Meeting is one confirmed contact as seen from Self: who it met, the impulse exchanged
// (zero when only detected), and the way Self left Other.
type Meeting struct {
	Self, Other uid.UID64
	Impact      float64
	Normal      geom.Vec
}

// Struck is what a plugin.Each behavior hosted here is told about an entity:
// which it is, and what it struck the tick before.
type Struck struct {
	ID       uid.UID64
	Contacts []Contact
}
