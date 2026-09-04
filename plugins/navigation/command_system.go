package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/selection"
)

// commandSystem issues move orders: a right-click gives every Selected
// entity a fresh MoveOrder/Path, unless the target is unreachable.
type commandSystem struct {
	pathFinder *pathFinder
	state      *CommandState

	query   *goke.Query
	cell    goke.Comp[board.Cell]
	orderID goke.CompID
}

var _ goke.System = (*commandSystem)(nil)

// newCommandSystem builds a commandSystem issuing move orders via pathFinder, driven by state.
func newCommandSystem(pathFinder *pathFinder, state *CommandState) *commandSystem {
	return &commandSystem{state: state, pathFinder: pathFinder}
}

func (s *commandSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell).Include(goke.Include[selection.Selected]()).Build()
	s.orderID = si.RegComp[MoveOrder]()
}

func (s *commandSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
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
