package effects

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

var _ goke.Module = (*module)(nil)

// module runs the effect system as one goke.Module.
type module struct {
	system   *effectSystem
	runnable goke.Runnable
}

// =================================================================
// goke.Module contract
// =================================================================

func (m *module) RegSystems(ecs *goke.ECS) { m.runnable = ecs.RegSys(m.system) }

func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.runnable, d)
	ctx.Sync()
}

// SetupSystems is empty — effects are cast at runtime.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types effects owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken { return []goke.CompToken{goke.LoadComp[Active]()} }
