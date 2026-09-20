package collisions

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// MaxContacts caps how many confirmed contacts one entity records in a tick,
// matching how many candidates the broad phase offers it in the first place.
//
// Extras past the cap are silently dropped, not evicted — and a dropped contact
// is one the entity never reacts to, bounce included.
const MaxContacts = MaxTouching

// Contact is one confirmed contact: who it was against, the impulse the two
// sides exchanged (zero when they were already separating, or when only
// sensed), and the way this entity leaves the other.
type Contact struct {
	Other  uid.UID64
	Impact float64
	Normal geom.Vec
}
