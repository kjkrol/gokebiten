package navigation

import (
	"cmp"
	"slices"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/selection"
	"github.com/kjkrol/uid"
)

// moveCommandSystem issues move orders: a right-click gives every Selected entity its own free
// cell at or around the target, nearest entity first; a Shift-click queues the target behind an
// order already in flight instead.
type moveCommandSystem struct {
	pathFinder *pathFinder
	state      *Resources

	query   *goke.Query
	cell    goke.Comp[board.Cell]
	order   goke.OptComp[MoveOrder]
	mover   goke.OptComp[board.Mover]
	orderID goke.CompID
}

var _ goke.System = (*moveCommandSystem)(nil)

// newMoveCommandSystem builds a moveCommandSystem issuing move orders via pathFinder.
func newMoveCommandSystem(pathFinder *pathFinder, state *Resources) *moveCommandSystem {
	return &moveCommandSystem{state: state, pathFinder: pathFinder}
}

func (s *moveCommandSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell).Optional(&s.order).Optional(&s.mover).Include(goke.Include[selection.Selected]()).Build()
	s.orderID = si.RegComp[MoveOrder]()
}

func (s *moveCommandSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
	if s.state.Pending == nil {
		return
	}
	cmd := *s.state.Pending
	target := cmd.Cell
	s.state.Pending = nil

	pf := s.pathFinder
	at := pf.terrain.Kind(target)

	var moves []pendingMove
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		cells := s.cell.Slice(cursor)
		orders := s.order.Slice(cursor)
		movers := s.mover.Slice(cursor)
		for i, id := range cursor.IDs {
			domain := board.DomainAt(movers, i)
			if !at.Admits(domain) {
				continue
			}
			if cmd.Append && orders != nil {
				orders[i].Enqueue(target)
				continue
			}
			m := pendingMove{id: id, from: cells[i].ID, domain: domain}
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
			path, ok = pf.findPath(m.id, m.domain, m.from, target)
		}
		if !ok {
			dest, path, ok = pf.nearestFree(m.id, m.domain, m.from, target, taken)
		}
		if !ok {
			continue
		}
		taken[dest] = true
		cb.AddOne(m.id, s.orderID, MoveOrder{Target: dest, Path: path, Leg: m.leg})
	}
}

// pendingMove is one Selected entity awaiting a destination.
type pendingMove struct {
	id     uid.UID64
	from   board.CellID
	leg    Leg
	domain board.Domain
}
