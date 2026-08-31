package board

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// Cell is an entity's logical position on the board.
type Cell struct{ ID CellID }

// MoveTo commands an entity to path toward Target until it arrives —
// SteeringSystem removes it (with Path) once Target is reached.
type MoveTo struct{ Target CellID }

// CellEntered is a one-tick tag added the tick an entity's Cell changes —
// query it to react to a unit stepping onto a cell.
type CellEntered struct{ ID CellID }

// SteeringSystem paths MoveTo-commanded entities toward their target,
// setting Velocity's direction and base speed toward the next waypoint.
type SteeringSystem struct {
	grid       Grid
	terrain    Terrain
	occupancy  Occupancy
	speed      int32
	space      *gokg.Space
	pathFinder *PathFinder

	query  *goke.Query
	cell   goke.Comp[Cell]
	pos    goke.Comp[world.Position]
	vel    goke.Comp[world.Velocity]
	moveTo goke.OptComp[MoveTo]
	path   goke.OptComp[Path]

	cellEnteredAdd goke.Comp[CellEntered]
	enterVM        *goke.ValueEditor
	arrivedEditor  *goke.Editor

	enteredQuery     *goke.Query
	cellEnteredClear goke.Comp[CellEntered]
	clearEditor      *goke.Editor

	sys goke.Runnable
}

var _ goke.Module = (*SteeringSystem)(nil)
var _ goke.System = (*SteeringSystem)(nil)
var _ gokebiten.PostLoader = (*SteeringSystem)(nil)

// arrivalEpsilon is how close (world-units) counts as "reached" a waypoint — small enough that the final snap is imperceptible.
const arrivalEpsilon = 2.0

// NewSteeringSystem builds a SteeringSystem whose base steering speed, before any world.SpeedModifier scales it, is speed world-units/sec.
func NewSteeringSystem(pathFinder *PathFinder, terrain Terrain, occupancy Occupancy, speed int32) *SteeringSystem {
	return &SteeringSystem{
		grid: pathFinder.Grid(), terrain: terrain, occupancy: occupancy, speed: speed,
		pathFinder: pathFinder,
	}
}

// BindSpace attaches the shared spatial index — arrivals snap to the cell center once bound; no-op (best-effort stop) if never called.
func (s *SteeringSystem) BindSpace(space *gokg.Space) { s.space = space }

func (s *SteeringSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell, &s.pos, &s.vel).
		Optional(&s.moveTo, &s.path).
		Build()
	s.arrivedEditor = s.query.NewEditorBuilder().Remove(goke.Remove[MoveTo](), goke.Remove[Path]()).Build()
	s.enterVM = s.query.NewValueEditorBuilder(&s.cellEnteredAdd).Build()

	s.enteredQuery = si.NewQueryBuilder(&s.cellEnteredClear).Build()
	s.clearEditor = s.enteredQuery.NewEditorBuilder().Remove(goke.Remove[CellEntered]()).Build()
}

func (s *SteeringSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
	s.clearEnteredTags(cb)

	snapped := false
	s.query.All()
	for s.query.Next() {
		cursor := s.query.Cursor()
		targets := s.moveTo.Slice(cursor)
		paths := s.path.Slice(cursor)
		if targets == nil {
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
			target := targets[i].Target
			p := &paths[i]
			previous := cells[i].ID
			actual, ok := s.grid.CellAt(center(positions[i]))
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
					p.Length = 0
				}
			}

			if (p.Length == 0 || p.Index >= p.Length) && actual != target {
				newPath, found := s.pathFinder.FindPath(s.terrain, s.occupancy, id, actual, target)
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
				_, passable := s.terrain.MovementCost(waypoint)
				if !passable || !s.occupancy.CanEnter(waypoint, id) {
					continue
				}
			}

			want := s.grid.CellCenter(waypoint)
			have := center(positions[i])
			dx, dy := want.X-have.X, want.Y-have.Y
			dist := math.Hypot(dx, dy)
			if dist > arrivalEpsilon {
				velocities[i].Dir = geom.NewVec(dx/dist, dy/dist)
				velocities[i].Value = s.speed
				continue
			}

			velocities[i].Value = 0

			if waypoint == target {
				if s.space != nil {
					ix, iy := int32(dx), int32(dy)
					if ix != 0 || iy != 0 {
						s.space.Translate(id, &positions[i].AABB, geom.NewVec(uint32(ix), uint32(iy)))
						snapped = true
					}
				}
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

func (s *SteeringSystem) clearEnteredTags(cb *goke.CmdBuf) {
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

func center(pos world.Position) geom.Vec[float64] {
	return geom.NewVec(float64(pos.TopLeft.X)+float64(pos.Size.X)/2, float64(pos.TopLeft.Y)+float64(pos.Size.Y)/2)
}

// RegSystems registers SteeringSystem itself as the per-tick system — see [goke.Module].
func (s *SteeringSystem) RegSystems(ecs *goke.ECS) {
	if s.sys == nil {
		s.sys = ecs.RegSys(s)
	}
}

// RunPlan runs SteeringSystem's Update for this tick — call from your own Game.Loop closure.
func (s *SteeringSystem) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(s.sys, d)
	ctx.Sync()
}

// SetupSystems is empty — spawning board entities is the game's responsibility.
func (s *SteeringSystem) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types SteeringSystem owns — see [goke.CompProvider].
func (s *SteeringSystem) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Cell](),
		goke.LoadComp[Path](),
		goke.LoadComp[MoveTo](),
		goke.LoadComp[CellEntered](),
	}
}

// PostLoad rebuilds Occupancy from every loaded entity's Cell component.
func (s *SteeringSystem) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var cell goke.Comp[Cell]
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
