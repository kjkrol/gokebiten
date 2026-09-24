package effects

import (
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/uid"
)

// Idle marks an entity whose last effect has just ended; it stays for one tick, so anything
// watching for it sees it whatever the order of the plugins' passes.
type Idle struct{}

// Idling is what an Each behavior hosted by effects gets, once, for an entity carrying Idle.
type Idling struct {
	ID   uid.UID64
	Base *world.Base
}
