package collisions

import (
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/uid"
)

// Collidable is the template an EntKind uses to take part in collisions: it
// writes the Collision component and tells space the entity CanCollide, so a
// broad-phase probe is offered it as a candidate.
//
// Adding Collision any other way leaves the index unaware of it, and the
// entity silently never collides.
func Collidable(space *gokg.Space) world.ComponentTemplate {
	return world.Const(Collision{}).WithEffect(func(_ Collision, id uid.UID64) {
		space.SetCapabilities(id, CanCollide)
	})
}
