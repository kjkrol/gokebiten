package collisions

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// Meeting is what a plugin.Between behavior hosted here is told: one confirmed
// contact as seen from Self — who it met, the impulse the two exchanged (zero
// when only detected), and the way Self left Other.
//
// A contact has no direction of its own, so a behavior runs for whichever way
// round its tags fit the pair: Between[Bullet, Target] always gets the bullet
// as Self.
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
