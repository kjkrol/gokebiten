package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/board"
)

// module registers and runs navigationSystem (always) and commandSystem
// (only when WithCommands enabled it) as one goke.Module.
type module struct {
	nav *navigationSystem
	cmd *commandSystem

	navRunnable goke.Runnable
	cmdRunnable goke.Runnable
}

var _ goke.Module = (*module)(nil)
var _ gokebiten.PostLoader = (*module)(nil)

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems registers nav (and cmd, if enabled) as the per-tick systems — see [goke.Module].
func (m *module) RegSystems(ecs *goke.ECS) {
	if m.navRunnable != nil {
		return
	}
	m.navRunnable = ecs.RegSys(m.nav)
	if m.cmd != nil {
		m.cmdRunnable = ecs.RegSys(m.cmd)
	}
}

// RunPlan runs nav's Update (and cmd's, if enabled) for this tick — call from your own Game.Loop closure.
func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.navRunnable, d)
	ctx.Sync()
	if m.cmdRunnable != nil {
		ctx.Run(m.cmdRunnable, d)
		ctx.Sync()
	}
}

// SetupSystems is empty — spawning board entities is the game's responsibility.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types navigation owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[board.Cell](),
		goke.LoadComp[MoveOrder](),
		goke.LoadComp[CellEntered](),
	}
}

// =================================================================
// gokebiten.PostLoader contract
// =================================================================

// PostLoad rebuilds board.Occupancy from every loaded entity's Cell component.
func (m *module) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var cell goke.Comp[board.Cell]
		query := si.NewQueryBuilder(&cell).Build()
		query.All()
		for query.Next() {
			cursor := query.Cursor()
			cells := cell.Slice(cursor)
			for i, id := range cursor.IDs {
				m.nav.occupancy.Enter(cells[i].ID, id)
			}
		}
	}}
}
