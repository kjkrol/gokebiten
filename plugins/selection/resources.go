package selection

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// PendingSelect is a completed click/drag gesture awaiting Update.
type PendingSelect struct {
	Start, End geom.Vec[int32]
	Additive   bool
}

// Resources is selection's single published Resources — live input state
// HandleEvents writes to and System reads/clears.
type Resources struct {
	Dragging    bool
	DragStart   geom.Vec[int32]
	DragCurrent geom.Vec[int32]
	Pending     *PendingSelect
	PendingIDs  []uid.UID64
}

func (*Resources) Resources() {}

// DragBox reports the screen-space rectangle of the drag gesture in progress, if any.
func (s *Resources) DragBox() (start, current geom.Vec[int32], dragging bool) {
	return s.DragStart, s.DragCurrent, s.Dragging
}
