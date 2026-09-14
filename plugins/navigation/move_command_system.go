package navigation

import (
	"cmp"
	"slices"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/selection"
	"github.com/kjkrol/uid"
)

// moveCommandSystem issues move orders: a right-click gives every Selected
// entity its own free cell at or around the target, nearest entity first.
type moveCommandSystem struct {
	pathFinder *pathFinder
	state      *Resources

	query   *goke.Query
	cell    goke.Comp[board.Cell]
	order   goke.OptComp[MoveOrder]
	orderID goke.CompID
}

var _ goke.System = (*moveCommandSystem)(nil)

// newMoveCommandSystem builds a moveCommandSystem issuing move orders via pathFinder, driven by state.
func newMoveCommandSystem(pathFinder *pathFinder, state *Resources) *moveCommandSystem {
	return &moveCommandSystem{state: state, pathFinder: pathFinder}
}

func (s *moveCommandSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell).Optional(&s.order).Include(goke.Include[selection.Selected]()).Build()
	s.orderID = si.RegComp[MoveOrder]()
}

func (s *moveCommandSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
	if s.state.PendingTarget == nil {
		return
	}
	target := *s.state.PendingTarget
	s.state.PendingTarget = nil

	pf := s.pathFinder
	if !pf.terrain.Kind(target).Passable {
		return
	}

	var moves []pendingMove
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		cells := s.cell.Slice(cursor)
		orders := s.order.Slice(cursor)
		for i, id := range cursor.IDs {
			m := pendingMove{id: id, from: cells[i].ID}
			if orders != nil && orders[i].Leg.Active {
				m.leg, m.from = orders[i].Leg, orders[i].Leg.To
			}
			moves = append(moves, m)
		}
	}
	slices.SortStableFunc(moves, func(a, b pendingMove) int {
		return cmp.Compare(pf.grid.Distance(a.from, target), pf.grid.Distance(b.from, target))
	})

	taken := make(map[board.CellID]bool)
	for n, m := range moves {
		dest := target
		var path Path
		ok := false
		if n == 0 {
			path, ok = pf.findPath(m.id, m.from, target)
		}
		if !ok {
			dest, path, ok = pf.nearestFree(m.id, m.from, target, taken)
		}
		if !ok {
			continue
		}
		taken[dest] = true
		cb.AddOne(m.id, s.orderID, MoveOrder{Target: dest, Path: path, Leg: m.leg})
	}
}

// pendingMove is one Selected entity awaiting a destination: where its next route starts, and the step it's finishing.
type pendingMove struct {
	id   uid.UID64
	from board.CellID
	leg  Leg
}
