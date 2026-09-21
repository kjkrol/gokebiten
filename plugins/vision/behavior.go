package vision

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// Sighting is one observer and everything in its view carrying the behavior's second tag,
// nearest first, possibly none. Steering is nil for an observer that cannot be steered.
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
