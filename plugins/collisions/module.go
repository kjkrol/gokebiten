package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
)

var _ goke.Module = (*module)(nil)

// module is the optional broad/narrow-phase collision engine, run over a
// *gokg.Space borrowed from world.Module — see Plugin.Install.
type module struct {
	space *gokg.Space
	ecs   *goke.ECS

	handlers     []CollisionHandler
	extra        []goke.System
	extraHandles []goke.Runnable
	hitDuration  time.Duration

	broadPhase  goke.Runnable
	narrowPhase goke.Runnable
	built       bool
}

// New builds the collision engine over space, expiring Hit tags after
// hitExpires by default - a touching entity's own HitExpires overrides
// it. Safe to call before ECS.Load, since systems register via
// RegSystems (see [goke.Module]).
func New(space *gokg.Space, ecs *goke.ECS, hitExpires time.Duration) *module {
	m := &module{space: space, ecs: ecs, hitDuration: hitExpires}
	m.extra = append(m.extra, NewTagExpirySystem(func(h *Hit) time.Time { return h.ExpiresAt() }))
	return m
}

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems builds and registers the collision systems — see [goke.Module].
func (m *module) RegSystems(ecs *goke.ECS) {
	if !m.built {
		m.build()
	}
}

func (m *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(m.broadPhase, d)
	ctx.Sync()

	ctx.Run(m.narrowPhase, d)
	ctx.Sync()

	for _, h := range m.extraHandles {
		ctx.Run(h, d)
		ctx.Sync()
	}
}

// SetupSystems is empty — the collision engine has no one-time seeding of its own.
func (m *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types the collision engine owns — see [goke.CompProvider].
func (m *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Collision](),
		goke.LoadComp[Hit](),
		goke.LoadComp[HitExpires](),
		goke.LoadComp[Sensor](),
		goke.LoadComp[Static](),
	}
}

// =================================================================
// collisions-specific
// =================================================================

func (m *module) SetCollisionHandlers(handlers ...CollisionHandler) *module {
	m.handlers = handlers
	return m
}

func (m *module) RegSys(sys goke.System) *module {
	m.extra = append(m.extra, sys)
	return m
}

func (m *module) build() {
	broad, narrow := m.useCollisions(MultiHandler(m.handlers...), m.hitDuration)
	m.broadPhase = m.ecs.RegSys(broad)
	m.narrowPhase = m.ecs.RegSys(narrow)

	for _, sys := range m.extra {
		m.extraHandles = append(m.extraHandles, m.ecs.RegSys(sys))
	}
	m.built = true
}

func (m *module) useCollisions(handler CollisionHandler, hitDuration time.Duration) (*BroadPhase, *NarrowPhase) {
	return NewBroadPhase(m.space), NewNarrowPhase(m.space, handler, hitDuration)
}
