package collision

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/uid"
)

// Collider is what makes an entity take part in collision detection — carrying it is
// all it takes — and holds what the narrow phase confirmed the entity struck.
type Collider struct {
	Struck      [MaxContacts]Contact
	StruckCount uint8

	// Indexed is the broad phase's own note that the index knows this entity; leave it false.
	Indexed bool
}

// Contacts is what this entity struck the tick before, in the order confirmed.
func (c *Collider) Contacts() []Contact { return c.Struck[:c.StruckCount] }

// addContact records one confirmed contact, up to MaxContacts.
func (c *Collider) addContact(other uid.UID64, impact float64, normal geom.Vec) {
	if c.StruckCount < MaxContacts {
		c.Struck[c.StruckCount] = Contact{Other: other, Impact: impact, Normal: normal}
		c.StruckCount++
	}
}

// clearContacts drops the previous tick's contacts.
func (c *Collider) clearContacts() { c.StruckCount = 0 }
