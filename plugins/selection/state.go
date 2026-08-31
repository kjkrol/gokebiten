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

// State is selection's live input state — HandleEvents implementations
// write to it, System reads/clears it. Published to Resources by Plugin,
// so a custom EventHandler can drive selection without touching System's
// internals.
type State struct {
	Dragging    bool
	DragStart   geom.Vec[int32]
	DragCurrent geom.Vec[int32]
	Pending     *PendingSelect
	PendingIDs  []uid.UID64
}

// DragBox reports the screen-space rectangle of the drag gesture in progress, if any.
func (s *State) DragBox() (start, current geom.Vec[int32], dragging bool) {
	return s.DragStart, s.DragCurrent, s.Dragging
}
