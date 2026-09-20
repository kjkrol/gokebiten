package vision

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// Sighting is what a plugin.Between behavior hosted here is told, once a tick
// for every observer carrying its first tag: the observer, and everything in
// its view that carries the second — nearest first, and empty when there is
// nothing of the kind to see.
//
// A sighting has a direction: Between[Predator, Prey] is a predator looking at
// prey, never the prey looking back. Steering is nil for an observer that
// cannot be steered; changing course goes through its Request. Seen belongs to
// the host and is only good until the behavior returns.
type Sighting struct {
	Self     uid.UID64
	Base     *world.Base
	Sight    *Sight
	Steering *world.Steering
	Seen     []Seen
}

// Seen is one entity in an observer's view: which, where and how it moves, how
// far off — and, through Carries, which of the tags the behavior declared.
type Seen struct {
	ID   uid.UID64
	Base *world.Base
	Dist float32
	plugin.TagSet
}
