package collisions

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// MaxTouching caps how many distinct neighbors Collision records in one
// tick — extras are silently dropped once full, not evicted.
const MaxTouching = 8

// Collision is what makes an entity take part in collision detection: who the broad
// phase found near it this tick, and what the narrow phase confirmed it struck.
type Collision struct {
	Touching      [MaxTouching]uid.UID64
	TouchingCount uint8
	Struck        [MaxContacts]Contact
	StruckCount   uint8
}

// Contacts is what this entity struck the tick before, in the order confirmed.
func (c *Collision) Contacts() []Contact { return c.Struck[:c.StruckCount] }

// addTouching records id as a neighbor this tick, deduplicated, up to
// MaxTouching — further neighbors past the cap are silently dropped.
func (c *Collision) addTouching(id uid.UID64) {
	for i := uint8(0); i < c.TouchingCount; i++ {
		if c.Touching[i] == id {
			return
		}
	}
	if c.TouchingCount < MaxTouching {
		c.Touching[c.TouchingCount] = id
		c.TouchingCount++
	}
}

// clear resets the touching list for the next tick.
func (c *Collision) clear() {
	c.TouchingCount = 0
}

// addContact records one confirmed contact, up to MaxContacts.
func (c *Collision) addContact(other uid.UID64, impact float64, normal geom.Vec) {
	if c.StruckCount < MaxContacts {
		c.Struck[c.StruckCount] = Contact{Other: other, Impact: impact, Normal: normal}
		c.StruckCount++
	}
}

// clearContacts drops the previous tick's contacts.
func (c *Collision) clearContacts() { c.StruckCount = 0 }
