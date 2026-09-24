package board

import (
	"github.com/kjkrol/gram/plugins/vision"
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.Module = (*module)(nil)

// module runs, every tick, the cell entities' Ground into the terrain, then the terrain bodies
// when the plugin was built WithCollision, then the standing report.
type module struct {
	cells    *cellEntitySystem
	standing *standingSystem
	bodies   *terrainBodySystem

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

// LoadComps lists the component types board writes, so a save loads without the vision plugin —
// see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	tokens := []goke.CompToken{goke.LoadComp[Cell](), goke.LoadComp[Mover](), goke.LoadComp[Ground]()}
	if m.bodies != nil {
		tokens = append(tokens, goke.LoadComp[vision.Transparency]())
	}
	return tokens
}
