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

// Contacts is what this entity actually struck, for game logic to read a tick
// later — give it to a kind with k.Const(collisions.Contacts{}).
type Contacts struct {
	Items [MaxContacts]Contact
	Count uint8
}

// All is what this entity struck, in the order the contacts were confirmed.
func (c *Contacts) All() []Contact { return c.Items[:c.Count] }

// add records one confirmed contact, up to MaxContacts.
func (c *Contacts) add(other uid.UID64, impact float64, normal geom.Vec) {
	if c.Count < MaxContacts {
		c.Items[c.Count] = Contact{Other: other, Impact: impact, Normal: normal}
		c.Count++
	}
}

// clear drops the previous tick's contacts.
func (c *Contacts) clear() { c.Count = 0 }
