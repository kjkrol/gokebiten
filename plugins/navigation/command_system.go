package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/selection"
)

// CommandSystem issues move orders: a right-click gives every Selected
// entity a fresh MoveOrder/Path, unless the target is unreachable.
type CommandSystem struct {
	terrain    board.Terrain
	occupancy  board.Occupancy
	pathFinder *PathFinder
	state      *CommandState

	query   *goke.Query
	cell    goke.Comp[board.Cell]
	orderID goke.CompID

	sys goke.Runnable
}

var _ goke.Module = (*CommandSystem)(nil)
var _ goke.System = (*CommandSystem)(nil)

// NewCommandSystem builds a CommandSystem issuing move orders via pathFinder, respecting terrain/occupancy, driven by state.
func NewCommandSystem(pathFinder *PathFinder, terrain board.Terrain, occupancy board.Occupancy, state *CommandState) *CommandSystem {
	return &CommandSystem{
		terrain: terrain, occupancy: occupancy, state: state,
		pathFinder: pathFinder,
	}
}

func (s *CommandSystem) Init(si *goke.SysInit) {
	s.query = si.NewQueryBuilder(&s.cell).Include(goke.Include[selection.Selected]()).Build()
	s.orderID = si.RegComp[MoveOrder]()
}

func (s *CommandSystem) Update(cb *goke.CmdBuf, _ time.Duration) {
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
			newPath, ok := s.pathFinder.FindPath(s.terrain, s.occupancy, id, cells[i].ID, target)
			if !ok {
				continue
			}
			cb.AddOne(id, s.orderID, MoveOrder{Target: target, Path: newPath})
		}
	}
}

// RegSystems registers CommandSystem itself as the per-tick system — see [goke.Module].
func (s *CommandSystem) RegSystems(ecs *goke.ECS) {
	if s.sys == nil {
		s.sys = ecs.RegSys(s)
	}
}

// RunPlan runs CommandSystem's Update for this tick — call from your own Game.Loop closure.
func (s *CommandSystem) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(s.sys, d)
	ctx.Sync()
}

// SetupSystems is empty — CommandSystem has no one-time seeding of its own.
func (s *CommandSystem) SetupSystems() []goke.System { return nil }

// LoadComps is empty — MoveOrder/Cell are already owned by NavigationSystem.
func (s *CommandSystem) LoadComps() []goke.CompToken { return nil }
