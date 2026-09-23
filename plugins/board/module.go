package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.Module = (*module)(nil)

// module runs the standing report every tick and, when the plugin was built WithCollision, the
// terrain bodies before it.
type module struct {
	standing *standingSystem
	bodies   *terrainBodies

	standingRunnable goke.Runnable
	bodiesRunnable   goke.Runnable
}

// =================================================================
// goke.Module contract
// =================================================================

func (m *module) RegSystems(ecs *goke.ECS) {
	if m.bodies != nil {
		m.bodiesRunnable = ecs.RegSys(m.bodies)
	}
	m.standingRunnable = ecs.RegSys(m.standing)
}

func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
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
	return []goke.CompToken{goke.LoadComp[Cell](), goke.LoadComp[Body](), goke.LoadComp[Mover]()}
}
