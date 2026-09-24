package collision

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/uid"
)

// Collider is what makes an entity take part in collision detection — carrying it is
// all it takes — and holds what the narrow phase confirmed the entity struck. Layers are the
// bits it collides on: two colliders touch only where their Layers share a bit, and zero is
// every layer — a board game uses its Domain bits, so a flyer and a walker pass through each other.
type Collider struct {
	Layers      uint8
	Struck      [MaxContacts]Contact
	StruckCount uint8
}

// touches reports whether the two colliders share a layer, zero standing for all of them.
func (c *Collider) touches(o *Collider) bool {
	return c.Layers == 0 || o.Layers == 0 || c.Layers&o.Layers != 0
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
