package navigation

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// MoveOrder commands an entity to path toward Target until it arrives —
// navigationSystem removes it once Target is reached.
type MoveOrder struct {
	Target board.CellID
	Path   Path
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
	speed      int32
	space      *gokg.Space
	pathFinder *pathFinder

	query *goke.Query
	cell  goke.Comp[board.Cell]
	pos   goke.Comp[world.Position]
	vel   goke.Comp[world.Velocity]
	order goke.OptComp[MoveOrder]

	cellEnteredAdd goke.Comp[CellEntered]
	enterVM        *goke.ValueEditor
	arrivedEditor  *goke.Editor

	enteredQuery     *goke.Query
	cellEnteredClear goke.Comp[CellEntered]
	clearEditor      *goke.Editor

	sys goke.Runnable
}

var _ goke.Module = (*navigationSystem)(nil)
var _ goke.System = (*navigationSystem)(nil)
var _ gokebiten.PostLoader = (*navigationSystem)(nil)

// arrivalEpsilon is how close (world-units) counts as "reached" a waypoint — small enough that the final snap is imperceptible.
const arrivalEpsilon = 2.0

// newNavigationSystem builds a navigationSystem whose base movement speed, before any world.SpeedModifier scales it, is speed world-units/sec.
func newNavigationSystem(pathFinder *pathFinder, grid board.Grid, terrain board.Terrain, occupancy board.Occupancy, speed int32) *navigationSystem {
	return &navigationSystem{
		grid: grid, terrain: terrain, occupancy: occupancy, speed: speed,
		pathFinder: pathFinder,
	}
}

// BindSpace attaches the shared spatial index — arrivals snap to the cell center once bound; no-op (best-effort stop) if never called.
func (s *navigationSystem) BindSpace(space *gokg.Space) { s.space = space }

func (s *navigationSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell, &s.pos, &s.vel).
		Optional(&s.order).
		Build()
	s.arrivedEditor = s.query.NewEditorBuilder().Remove(goke.Remove[MoveOrder]()).Build()
	s.enterVM = s.query.NewValueEditorBuilder(&s.cellEnteredAdd).Build()

	s.enteredQuery = si.NewQueryBuilder(&s.cellEnteredClear).Build()
	s.clearEditor = s.enteredQuery.NewEditorBuilder().Remove(goke.Remove[CellEntered]()).Build()
}

func (s *navigationSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
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
		positions := s.pos.Slice(cursor)
		velocities := s.vel.Slice(cursor)
		snap := s.query.ChunkSnapshot()

		var enteredIDs []uid.UID64
		var enteredVals []CellEntered
		var arrivedIDs []uid.UID64

		for i, id := range cursor.IDs {
			target := orders[i].Target
			p := &orders[i].Path
			previous := cells[i].ID
			actual, ok := s.grid.CellAt(board.Center(positions[i]))
			if !ok {
				actual = previous
			}

			if actual != previous {
				s.occupancy.Leave(previous, id)
				s.occupancy.Enter(actual, id)
				cells[i].ID = actual
				enteredIDs = append(enteredIDs, id)
				enteredVals = append(enteredVals, CellEntered{ID: actual})

				expected := target
				if p.Length > 0 && p.Index < p.Length {
					expected = p.Steps[p.Index]
				}
				if actual != expected {
					c1, c2, diag := s.grid.DiagonalNeighbors(previous, expected)
					isFlanker := diag && (actual == c1 || actual == c2)
					if !isFlanker {
						p.Length = 0
					}
				}
			}

			if (p.Length == 0 || p.Index >= p.Length) && actual != target {
				newPath, found := s.pathFinder.findPath(id, actual, target)
				if !found {
					continue
				}
				*p = newPath
			}

			waypoint := target
			if p.Length > 0 && p.Index < p.Length {
				waypoint = p.Steps[p.Index]
			}

			if actual != waypoint {
				if !s.terrain.Kind(waypoint).Passable || !s.occupancy.CanEnter(waypoint, id) {
					continue
				}
			}

			want := s.grid.CellCenter(waypoint)
			have := board.Center(positions[i])
			dx, dy := want.X-have.X, want.Y-have.Y
			if s.space != nil {
				dx = shortestAxisDelta(have.X, want.X, s.space.Width, s.space.Toroidal)
				dy = shortestAxisDelta(have.Y, want.Y, s.space.Height, s.space.Toroidal)
			}
			dist := math.Hypot(dx, dy)
			if dist > arrivalEpsilon {
				velocities[i].Dir = geom.NewVec(dx/dist, dy/dist)
				velocities[i].Value = s.speed
				continue
			}

			velocities[i].Value = 0

			if s.space != nil {
				ix, iy := int32(dx), int32(dy)
				if ix != 0 || iy != 0 {
					s.space.Translate(id, &positions[i].AABB, geom.NewVec(uint32(ix), uint32(iy)))
					snapped = true
				}
			}

			if waypoint == target {
				arrivedIDs = append(arrivedIDs, id)
				continue
			}

			p.Index++
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

func shortestAxisDelta(have, want float64, size uint32, toroidal bool) float64 {
	d := want - have
	if !toroidal || size == 0 {
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

// RegSystems registers navigationSystem itself as the per-tick system — see [goke.Module].
func (s *navigationSystem) RegSystems(ecs *goke.ECS) {
	if s.sys == nil {
		s.sys = ecs.RegSys(s)
	}
}

// RunPlan runs navigationSystem's Update for this tick — call from your own Game.Loop closure.
func (s *navigationSystem) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(s.sys, d)
	ctx.Sync()
}

// SetupSystems is empty — spawning board entities is the game's responsibility.
func (s *navigationSystem) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types navigationSystem owns — see [goke.CompProvider].
func (s *navigationSystem) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[board.Cell](),
		goke.LoadComp[MoveOrder](),
		goke.LoadComp[CellEntered](),
	}
}

// PostLoad rebuilds board.Occupancy from every loaded entity's Cell component.
func (s *navigationSystem) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var cell goke.Comp[board.Cell]
		query := si.NewQueryBuilder(&cell).Build()
		query.All()
		for query.Next() {
			cursor := query.Cursor()
			cells := cell.Slice(cursor)
			for i, id := range cursor.IDs {
				s.occupancy.Enter(cells[i].ID, id)
			}
		}
	}}
}
