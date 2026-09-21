package vision

import (
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
)

var _ goke.Module = (*module)(nil)

// module registers ScanSystem as vision's single per-tick system.
type module struct {
	sys      *ScanSystem
	runnable goke.Runnable
}

func newModule(space *aabbworld.Space, host *plugin.PairHost[Sighting]) *module {
	return &module{sys: newScanSystem(space, host)}
}

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems registers the scan as the per-tick system — see [goke.Module].
func (m *module) RegSystems(ecs *goke.ECS) {
	if m.runnable != nil {
		return
	}
	m.runnable = ecs.RegSys(m.sys)
}

// RunPlan runs the scan for this tick — call from your own Game.Loop closure.
func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.runnable, d)
	ctx.Sync()
}

// SetupSystems is empty — vision has no one-time seeding of its own.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types vision owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Sight](),
		goke.LoadComp[SightOutline](),
	}
}
