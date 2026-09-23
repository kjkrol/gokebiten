package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.Module = (*module)(nil)

// module runs terrainBodies when the plugin was built WithCollision.
type module struct {
	bodies   *terrainBodies
	runnable goke.Runnable
}

// =================================================================
// goke.Module contract
// =================================================================

func (m *module) RegSystems(ecs *goke.ECS) { m.runnable = ecs.RegSys(m.bodies) }

func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.runnable, d)
	ctx.Sync()
}

// SetupSystems is empty — the bodies build themselves in their own Init.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types board owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken { return []goke.CompToken{goke.LoadComp[Body]()} }
