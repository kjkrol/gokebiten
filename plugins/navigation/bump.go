package navigation

import (
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/collision"
)

// bumped is the Struck behavior navigation registers on collision: an entity under orders that
// struck someone is marked Bumped, unless it is still holding the route a bump gave it.
func bumped() plugin.Behavior {
	return collision.Each[MoveOrder](func(_ plugin.Tick, o *MoveOrder, s collision.Struck) {
		if len(s.Contacts) > 0 && o.Cooldown == 0 {
			o.Bumped = true
		}
	})
}
