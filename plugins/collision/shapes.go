package collision

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// Contactee is one side of a contact as a ShapeTest sees it.
type Contactee struct {
	ID   uid.UID64
	Base *world.Base
}

// ShapeTest says whether two entities whose boxes overlap really touch, and by how much.
type ShapeTest func(t plugin.Tick, a, b Contactee, pen geom.Vec) (geom.Vec, bool)

// BoxesTouch is the ShapeTest an unset one amounts to, at no cost: overlapping boxes touch.
func BoxesTouch(_ plugin.Tick, _, _ Contactee, pen geom.Vec) (geom.Vec, bool) { return pen, true }
