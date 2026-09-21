package selection

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/control"
)

// DefaultEventHandler turns left-click and left-drag input into Resources.
type DefaultEventHandler struct{ state *Resources }

var _ control.EventHandler = (*DefaultEventHandler)(nil)

// NewDefaultEventHandler builds a DefaultEventHandler writing into state.
func NewDefaultEventHandler(state *Resources) *DefaultEventHandler {
	return &DefaultEventHandler{state: state}
}

func (h *DefaultEventHandler) HandleEvents(events *control.InputEvents) {
	s := h.state
	if s.Dragging {
		s.DragCurrent = events.MousePos
	}
	for _, c := range events.ClickQueue {
		if c.Button != ebiten.MouseButtonLeft {
			continue
		}
		switch c.Action {
		case control.ActionPress:
			s.Dragging = true
			s.DragStart = c.Pos
			s.DragCurrent = c.Pos
		case control.ActionRelease:
			if !s.Dragging {
				continue
			}
			s.Dragging = false
			s.Pending = &PendingSelect{Start: s.DragStart, End: c.Pos, Additive: events.Modifiers.Shift}
		}
	}
}
