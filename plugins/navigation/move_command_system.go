package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/selection"
)

// moveCommandSystem issues move orders: a right-click gives every Selected
// entity a fresh MoveOrder/Path, unless the target is unreachable.
type moveCommandSystem struct {
	pathFinder *pathFinder
	state      *CommandState

	query   *goke.Query
	cell    goke.Comp[board.Cell]
	orderID goke.CompID
}

var _ goke.System = (*moveCommandSystem)(nil)

// newMoveCommandSystem builds a moveCommandSystem issuing move orders via pathFinder, driven by state.
func newMoveCommandSystem(pathFinder *pathFinder, state *CommandState) *moveCommandSystem {
	return &moveCommandSystem{state: state, pathFinder: pathFinder}
}

func (s *moveCommandSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell).Include(goke.Include[selection.Selected]()).Build()
	s.orderID = si.RegComp[MoveOrder]()
}

func (s *moveCommandSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
	if s.state.PendingTarget == nil {
		return
	}
	target := *s.state.PendingTarget
	s.state.PendingTarget = nil

	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		cells := s.cell.Slice(cursor)
		for i, id := range cursor.IDs {
			newPath, ok := s.pathFinder.findPath(id, cells[i].ID, target)
			if !ok {
				continue
			}
			cb.AddOne(id, s.orderID, MoveOrder{Target: target, Path: newPath})
		}
	}
}
