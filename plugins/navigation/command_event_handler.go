package navigation

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

// DefaultCommandEventHandler turns a right-click into a move-order target —
// the default control.EventHandler for a navigation Plugin with WithCommands.
// Write your own against the same CommandState for a different binding scheme.
type DefaultCommandEventHandler struct {
	grid   board.Grid
	camera render.Camera
	state  *CommandState
}

var _ control.EventHandler = (*DefaultCommandEventHandler)(nil)

// NewDefaultCommandEventHandler builds a DefaultCommandEventHandler writing into state.
func NewDefaultCommandEventHandler(grid board.Grid, camera render.Camera, state *CommandState) *DefaultCommandEventHandler {
	return &DefaultCommandEventHandler{grid: grid, camera: camera, state: state}
}

func (h *DefaultCommandEventHandler) HandleEvents(events *control.InputEvents) {
	for _, c := range events.ClickQueue {
		if c.Button != ebiten.MouseButtonRight || c.Action != control.ActionPress {
			continue
		}
		wx, wy := h.camera.FromScreen(float32(c.Pos.X), float32(c.Pos.Y))
		target, ok := h.grid.CellAt(geom.NewVec(float64(wx), float64(wy)))
		if !ok {
			continue
		}
		h.state.PendingTarget = &target
	}
}
