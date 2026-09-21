package navigation

import (
	"math"
	"slices"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// MoveOrder commands an entity to path toward Target until it arrives —
// navigationSystem removes it once Target is reached.
type MoveOrder struct {
	Target board.CellID
	Path   Path
	Leg    Leg
	Waited time.Duration
}

// Leg is the single step an entity is travelling: every cell it holds in
// Occupancy until it reaches To's center.
type Leg struct {
	From, To board.CellID
	C1, C2   board.CellID
	Diagonal bool
	Active   bool
}

// cells lists every cell leg holds: From, To, and both corners of a diagonal step.
func (l Leg) cells() []board.CellID {
	if l.Diagonal {
		return []board.CellID{l.From, l.To, l.C1, l.C2}
	}
	return []board.CellID{l.From, l.To}
}

// CellEntered is a one-tick tag added the tick an entity's Cell changes —
// query it to react to a unit stepping onto a cell.
type CellEntered struct{ ID board.CellID }

// navigationSystem paths MoveOrder-commanded entities toward their target,
// setting Velocity's direction and base speed toward the next waypoint.
type navigationSystem struct {
	grid       board.Grid
	terrain    board.Terrain
	occupancy  board.Occupancy
	speed      float64
	space      *aabbworld.Space
	pathFinder *pathFinder

	query *goke.Query
	cell  goke.Comp[board.Cell]
	base  goke.Comp[world.Base]
	order goke.OptComp[MoveOrder]

	cellEnteredAdd goke.Comp[CellEntered]
	enterVM        *goke.ValueEditor
	arrivedEditor  *goke.Editor
	arrivedVM      *goke.ValueEditor

	enteredQuery     *goke.Query
	cellEnteredClear goke.Comp[CellEntered]
	clearEditor      *goke.Editor
}

var _ goke.System = (*navigationSystem)(nil)

// targetWaitTimeout is how long an entity waits for an occupied target before settling nearby.
const targetWaitTimeout = 500 * time.Millisecond

// arrivalEpsilon is how close, in world units, counts as having reached a waypoint.
const arrivalEpsilon = 2.0

// newNavigationSystem builds a navigationSystem moving entities at speed world units a second.
func newNavigationSystem(pathFinder *pathFinder, grid board.Grid, terrain board.Terrain, occupancy board.Occupancy, speed float64) *navigationSystem {
	return &navigationSystem{
		grid: grid, terrain: terrain, occupancy: occupancy, speed: speed,
		pathFinder: pathFinder,
	}
}

// BindSpace attaches the shared spatial index, so arrivals snap to the cell centre.
func (s *navigationSystem) BindSpace(space *aabbworld.Space) { s.space = space }

func (s *navigationSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell, &s.base).
		Optional(&s.order).
		Build()
	s.arrivedEditor = s.query.NewEditorBuilder().Remove(goke.Remove[MoveOrder]()).Build()
	s.enterVM = s.query.NewValueEditorBuilder(&s.cellEnteredAdd).Build()
	s.arrivedVM = s.query.NewValueEditorBuilder(&s.cellEnteredAdd).
		Remove(goke.Remove[MoveOrder]()).Build()

	s.enteredQuery = si.NewQueryBuilder(&s.cellEnteredClear).Build()
	s.clearEditor = s.enteredQuery.NewEditorBuilder().Remove(goke.Remove[CellEntered]()).Build()
}

func (s *navigationSystem) Update(cb *goke.CmdBuf, d time.Duration) {
	s.clearEnteredTags(cb)

	snapped := false
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		orders := s.order.Slice(cursor)
		if orders == nil {
			continue
		}

		cells := s.cell.Slice(cursor)
		bases := s.base.Slice(cursor)
		snap := s.query.ChunkSnapshot()

		var enteredIDs []uid.UID64
		var enteredVals []CellEntered
		var arrivedIDs []uid.UID64
		var bothIDs []uid.UID64
		var bothVals []CellEntered

		for i, id := range cursor.IDs {
			target := orders[i].Target
			p := &orders[i].Path
			leg := &orders[i].Leg
			current := cells[i].ID
			actual, ok := s.grid.CellAt(board.Center(bases[i].Pos))
			if !ok {
				actual = current
			}

			entered := false
			moveTo := func(c board.CellID) {
				if c == cells[i].ID {
					return
				}
				cells[i].ID = c
				if entered {
					enteredVals[len(enteredVals)-1] = CellEntered{ID: c}
					return
				}
				entered = true
				enteredIDs = append(enteredIDs, id)
				enteredVals = append(enteredVals, CellEntered{ID: c})
			}

			switch {
			case leg.Active && !slices.Contains(leg.cells(), actual):
				s.releaseLeg(*leg, id)
				s.occupancy.Enter(actual, id)
				moveTo(actual)
				*leg = Leg{}
				p.Length = 0
			case leg.Active && (actual == leg.From || actual == leg.To):
				moveTo(actual)
			case !leg.Active && actual != current:
				s.occupancy.Leave(current, id)
				s.occupancy.Enter(actual, id)
				moveTo(actual)
				p.Length = 0
			}

			if !leg.Active && (p.Length == 0 || p.Index >= p.Length) && cells[i].ID != target {
				newPath, found := s.pathFinder.findPath(id, cells[i].ID, target)
				if !found {
					bases[i].Vel.Value = 0
					orders[i].Waited += d
					if orders[i].Waited < targetWaitTimeout {
						continue
					}
					dest, destPath, ok := s.pathFinder.nearestFree(id, cells[i].ID, target, nil)
					if !ok {
						arrivedIDs = append(arrivedIDs, id)
						continue
					}
					orders[i].Target, target = dest, dest
					newPath = destPath
				}
				orders[i].Waited = 0
				*p = newPath
			}

			waypoint := target
			switch {
			case leg.Active:
				waypoint = leg.To
			case p.Length > 0 && p.Index < p.Length:
				waypoint = p.Steps[p.Index]
			}

			if !leg.Active && waypoint != cells[i].ID {
				reserved, ok := s.reserveLeg(cells[i].ID, waypoint, id)
				if !ok {
					bases[i].Vel.Value = 0
					p.Length = 0
					continue
				}
				*leg = reserved
			}

			want := s.grid.CellCenter(waypoint)
			have := board.Center(bases[i].Pos)
			dx, dy := want.X-have.X, want.Y-have.Y
			if s.space != nil {
				dx = shortestAxisDelta(have.X, want.X, s.space.Width, s.space.Edges.WrapsX())
				dy = shortestAxisDelta(have.Y, want.Y, s.space.Height, s.space.Edges.WrapsY())
			}
			dist := math.Hypot(dx, dy)
			if dist > arrivalEpsilon {
				bases[i].Vel.Dir = geom.NewVec(dx/dist, dy/dist)
				bases[i].Vel.Value = s.speed
				continue
			}

			bases[i].Vel.Value = 0

			if s.space != nil && (dx != 0 || dy != 0) {
				s.space.Translate(id, &bases[i].Pos.AABB, geom.NewVec(dx, dy))
				snapped = true
			}

			if leg.Active {
				s.releaseLeg(*leg, id)
				s.occupancy.Enter(leg.To, id)
				moveTo(leg.To)
				*leg = Leg{}
			}

			if p.Index < p.Length && p.Steps[p.Index] == waypoint {
				p.Index++
			}

			if waypoint == target {
				if entered {
					n := len(enteredIDs) - 1
					bothIDs = append(bothIDs, enteredIDs[n])
					bothVals = append(bothVals, enteredVals[n])
					enteredIDs, enteredVals = enteredIDs[:n], enteredVals[:n]
				} else {
					arrivedIDs = append(arrivedIDs, id)
				}
			}
		}

		if len(bothIDs) > 0 {
			vals := cb.AddCompValue(s.arrivedVM, &s.cellEnteredAdd, snap, bothIDs)
			copy(vals, bothVals)
		}
		if len(enteredIDs) > 0 {
			vals := cb.AddCompValue(s.enterVM, &s.cellEnteredAdd, snap, enteredIDs)
			copy(vals, enteredVals)
		}
		if len(arrivedIDs) > 0 {
			buf := s.query.BeginMigrate(cb)
			for _, id := range arrivedIDs {
				buf.Add(id)
			}
			buf.Commit(s.arrivedEditor)
		}
	}

	if snapped {
		s.space.Flush(nil)
	}
}

// reserveLeg claims every cell a step from→to can touch, or none of them and false.
func (s *navigationSystem) reserveLeg(from, to board.CellID, id uid.UID64) (Leg, bool) {
	leg := Leg{From: from, To: to, Active: true}
	if c1, c2, diag := s.grid.DiagonalNeighbors(from, to); diag {
		leg.C1, leg.C2, leg.Diagonal = c1, c2, true
	}
	for _, c := range leg.cells()[1:] {
		if !s.terrain.Kind(c).Passable || !s.occupancy.CanEnter(c, id) {
			return Leg{}, false
		}
	}
	for _, c := range leg.cells() {
		s.occupancy.Enter(c, id)
	}
	return leg, true
}

// releaseLeg gives up every cell leg holds.
func (s *navigationSystem) releaseLeg(leg Leg, id uid.UID64) {
	for _, c := range leg.cells() {
		s.occupancy.Leave(c, id)
	}
}

func shortestAxisDelta(have, want float64, size uint32, wraps bool) float64 {
	d := want - have
	if !wraps || size == 0 {
		return d
	}
	s := float64(size)
	if d > s/2 {
		d -= s
	} else if d < -s/2 {
		d += s
	}
	return d
}

func (s *navigationSystem) clearEnteredTags(cb *goke.CmdBuf) {
	s.enteredQuery.All()
	for s.enteredQuery.Next() {
		cursor := s.enteredQuery.Cursor()
		buf := s.enteredQuery.BeginMigrate(cb)
		for _, id := range cursor.IDs {
			buf.Add(id)
		}
		buf.Commit(s.clearEditor)
	}
}
