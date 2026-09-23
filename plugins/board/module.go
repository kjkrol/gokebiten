package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.Module = (*module)(nil)

// module runs, every tick, the cell entities' Ground into the terrain, then the terrain bodies
// when the plugin was built WithCollision, then the standing report.
type module struct {
	cells    *cellEntities
	standing *standingSystem
	bodies   *terrainBodies

	cellsRunnable    goke.Runnable
	standingRunnable goke.Runnable
	bodiesRunnable   goke.Runnable
}

// =================================================================
// goke.Module contract
// =================================================================

func (m *module) RegSystems(ecs *goke.ECS) {
	m.cellsRunnable = ecs.RegSys(m.cells)
	if m.bodies != nil {
		m.bodiesRunnable = ecs.RegSys(m.bodies)
	}
	m.standingRunnable = ecs.RegSys(m.standing)
}

func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.cellsRunnable, d)
	if m.bodies != nil {
		ctx.Run(m.bodiesRunnable, d)
	}
	ctx.Run(m.standingRunnable, d)
	ctx.Sync()
}

// SetupSystems is empty — the bodies build themselves in their own Init.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types board owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{goke.LoadComp[Cell](), goke.LoadComp[Mover](), goke.LoadComp[Ground]()}
}
