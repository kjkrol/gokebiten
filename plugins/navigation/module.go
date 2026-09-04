package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/board"
)

// module registers and runs navigationSystem (always) and moveCommandSystem
// (only when WithCommands enabled it) as one goke.Module.
type module struct {
	navigationSystem  *navigationSystem
	moveCommandSystem *moveCommandSystem

	navSysRunnable     goke.Runnable
	moveCmdSysRunnable goke.Runnable
}

var _ goke.Module = (*module)(nil)
var _ gokebiten.PostLoader = (*module)(nil)

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems registers nav (and cmd, if enabled) as the per-tick systems — see [goke.Module].
func (m *module) RegSystems(ecs *goke.ECS) {
	m.navSysRunnable = ecs.RegSys(m.navigationSystem)
	m.moveCmdSysRunnable = ecs.RegSys(m.moveCommandSystem)

}

// RunPlan runs nav's Update (and cmd's, if enabled) for this tick — call from your own Game.Loop closure.
func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.navSysRunnable, d)
	ctx.Sync()
	ctx.Run(m.moveCmdSysRunnable, d)
	ctx.Sync()

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
				m.navigationSystem.occupancy.Enter(cells[i].ID, id)
			}
		}
	}}
}
