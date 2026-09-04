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

// New builds the collision engine over space — safe to call before ECS.Load, since systems register via RegSystems (see [goke.Module]).
func New(space *gokg.Space, ecs *goke.ECS) *module {
	return &module{space: space, ecs: ecs}
}

// =================================================================
// goke.Module contract
// =================================================================

// RegSystems builds and registers the collision systems — see [goke.Module].
func (p *module) RegSystems(ecs *goke.ECS) {
	if !p.built {
		p.build()
	}
}

func (p *module) RunPlan(ctx goke.RunCtx, d time.Duration) {
	ctx.Run(p.broadPhase, d)
	ctx.Sync()

	ctx.Run(p.narrowPhase, d)
	ctx.Sync()

	for _, h := range p.extraHandles {
		ctx.Run(h, d)
		ctx.Sync()
	}
}

// SetupSystems is empty — the collision engine has no one-time seeding of its own.
func (p *module) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types the collision engine owns — see [goke.CompProvider].
func (p *module) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Collision](),
		goke.LoadComp[Hit](),
		goke.LoadComp[Sensor](),
		goke.LoadComp[Static](),
	}
}

// =================================================================
// collisions-specific
// =================================================================

func (p *module) SetCollisionHandlers(handlers ...CollisionHandler) *module {
	p.handlers = handlers
	return p
}

func (p *module) SetHitExpires(duration time.Duration) *module {
	p.hitDuration = duration
	p.extra = append(p.extra, NewTagExpirySystem(func(h *Hit) time.Time { return h.ExpiresAt() }))
	return p
}

func (p *module) RegSys(sys goke.System) *module {
	p.extra = append(p.extra, sys)
	return p
}

func (p *module) build() {
	broad, narrow := p.useCollisions(MultiHandler(p.handlers...), p.hitDuration)
	p.broadPhase = p.ecs.RegSys(broad)
	p.narrowPhase = p.ecs.RegSys(narrow)

	for _, sys := range p.extra {
		p.extraHandles = append(p.extraHandles, p.ecs.RegSys(sys))
	}
	p.built = true
}

func (p *module) useCollisions(handler CollisionHandler, hitDuration time.Duration) (*BroadPhase, *NarrowPhase) {
	return NewBroadPhase(p.space), NewNarrowPhase(p.space, handler, hitDuration)
}
