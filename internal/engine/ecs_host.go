package engine

import (
	"reflect"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/render"
)

// ecsHost owns the full lifecycle of one *goke.ECS instance: queuing
// ctx.Use/ctx.Track/UseModule/Setup work until a single ecs.Setup call, and
// the tracked-value bookkeeping Persistence.Save/Load need. Engine builds a
// fresh ecsHost each time it enters a Stage — see stageRuntime/enterStage.
type ecsHost struct {
	ecs       *goke.ECS
	resources *storage

	tracked      []any
	pendingSetup []func() []goke.System
	names        map[string]bool
}

func newECSHost() *ecsHost {
	return &ecsHost{ecs: goke.New(), resources: newStorage()}
}

// track records v so providedComps/postLoadSystems/runRestore/saveTargets
// can find it later.
func (h *ecsHost) track(v any) { h.tracked = append(h.tracked, v) }

// addPendingSetup queues producer to run once, during flushPendingSetup.
func (h *ecsHost) addPendingSetup(producer func() []goke.System) {
	h.pendingSetup = append(h.pendingSetup, producer)
}

func (h *ecsHost) useModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(si *goke.SysInit) { m.RegSystems(h.ecs) }}
	h.track(m)
	h.addPendingSetup(func() []goke.System { return append(m.SetupSystems(), regSys) })
}

func (h *ecsHost) setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		h.track(p)
		h.addPendingSetup(p.SetupSystems)
	}
}

func (h *ecsHost) regSys(factory func() goke.System) goke.Runnable {
	return h.ecs.RegSys(factory())
}

func (h *ecsHost) registerRenderer(factory func() render.Renderer) render.Renderer {
	r := factory()

	sys := goke.SystemFn{OnInit: func(si *goke.SysInit) { r.Init(si) }}
	h.addPendingSetup(func() []goke.System { return []goke.System{sys} })

	return r
}

// providedComps collects LoadComps from every tracked value implementing goke.CompProvider.
func (h *ecsHost) providedComps() []goke.CompToken { return goke.ProvidedComps(h.tracked...) }

// postLoadSystems collects PostLoad from every tracked value implementing PostLoader.
func (h *ecsHost) postLoadSystems() []goke.System {
	var systems []goke.System
	for _, v := range h.tracked {
		if pl, ok := v.(plugin.PostLoader); ok {
			systems = append(systems, pl.PostLoad())
		}
	}
	return systems
}

// runRestore calls Restore on every tracked value implementing Restorer,
// synchronously, right after Persistence.Load decodes their Persisted() pointers.
func (h *ecsHost) runRestore() {
	for _, v := range h.tracked {
		if r, ok := v.(plugin.Restorer); ok {
			r.Restore()
		}
	}
}

// runPopulate calls Populate on every tracked value implementing Populator, stopping at the first error.
func (h *ecsHost) runPopulate() error {
	for _, v := range h.tracked {
		if p, ok := v.(plugin.Populator); ok {
			if err := p.Populate(); err != nil {
				return err
			}
		}
	}
	return nil
}

// saveTargets collects Persisted from every tracked value implementing
// Serializable, keyed by its Go type name (tracked values have no Plugin.Name()).
func (h *ecsHost) saveTargets() map[string][]any {
	out := make(map[string][]any)
	for _, v := range h.tracked {
		if s, ok := v.(plugin.Serializable); ok {
			out[reflect.TypeOf(v).String()] = s.Persisted()
		}
	}
	return out
}

// persistGroups combines tracked Serializables, plugin-published
// Serializables, and extra into one name-keyed map for save/load.
func (h *ecsHost) persistGroups(extra ...any) map[string][]any {
	groups := h.saveTargets()
	for name, targets := range h.resources.persisted() {
		groups[name] = targets
	}
	for _, r := range extra {
		groups[reflect.TypeOf(r).String()] = []any{r}
	}
	return groups
}

// flushPendingSetup evaluates every deferred producer once and runs the result through a single ecs.Setup call.
func (h *ecsHost) flushPendingSetup() {
	if len(h.pendingSetup) == 0 {
		return
	}
	var systems []goke.System
	for _, produce := range h.pendingSetup {
		systems = append(systems, produce()...)
	}
	h.ecs.Setup(systems...)
	h.pendingSetup = nil
}
