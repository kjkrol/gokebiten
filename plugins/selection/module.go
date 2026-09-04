package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

// module registers SelectionSystem as selection's single per-tick system.
type module struct {
	sys      *SelectionSystem
	runnable goke.Runnable
}

var _ goke.Module = (*module)(nil)

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems registers sys as the per-tick system — see [goke.Module].
func (m *module) RegSystems(ecs *goke.ECS) {
	if m.runnable != nil {
		return
	}
	m.runnable = ecs.RegSys(m.sys)
}

// RunPlan runs sys's Update for this tick — call from your own Game.Loop closure.
func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.runnable, d)
	ctx.Sync()
}

// SetupSystems is empty — selection has no one-time seeding of its own.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types selection owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{goke.LoadComp[Selected]()}
}
