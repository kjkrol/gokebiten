package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
)

var _ goke.Module = (*Collisions)(nil)

// Collisions is the optional broad/narrow-phase collision engine, run over a
// *gokg.Space borrowed from world.Module — see Plugin.Install.
type Collisions struct {
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

// New builds Collisions over space — safe to call before ECS.Load, since systems register lazily on first RunPlan/RegSystems.
func New(space *gokg.Space, ecs *goke.ECS) *Collisions {
	return &Collisions{space: space, ecs: ecs}
}

func (p *Collisions) SetCollisionHandlers(handlers ...CollisionHandler) *Collisions {
	p.handlers = handlers
	return p
}

func (p *Collisions) SetHitExpires(duration time.Duration) *Collisions {
	p.hitDuration = duration
	p.extra = append(p.extra, NewTagExpirySystem(func(h *Hit) time.Time { return h.ExpiresAt() }))
	return p
}

func (p *Collisions) RegSys(sys goke.System) *Collisions {
	p.extra = append(p.extra, sys)
	return p
}

// RegSystems builds and registers the collision systems — see [goke.Module]; prefer RunPlan, which calls this lazily at the right time.
func (p *Collisions) RegSystems(ecs *goke.ECS) {
	if !p.built {
		p.build()
	}
}

func (p *Collisions) RunPlan(ctx goke.RunCtx, d time.Duration) {
	if !p.built {
		p.build()
	}
	ctx.Run(p.broadPhase, d)
	ctx.Sync()

	ctx.Run(p.narrowPhase, d)
	ctx.Sync()

	for _, h := range p.extraHandles {
		ctx.Run(h, d)
		ctx.Sync()
	}
}

// SetupSystems is empty — Collisions has no one-time seeding of its own.
func (p *Collisions) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types collisions owns — see [goke.CompProvider].
func (p *Collisions) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[Collision](),
		goke.LoadComp[Hit](),
		goke.LoadComp[Sensor](),
		goke.LoadComp[Static](),
	}
}

func (p *Collisions) build() {
	broad, narrow := p.useCollisions(MultiHandler(p.handlers...), p.hitDuration)
	p.broadPhase = p.ecs.RegSys(broad)
	p.narrowPhase = p.ecs.RegSys(narrow)

	for _, sys := range p.extra {
		p.extraHandles = append(p.extraHandles, p.ecs.RegSys(sys))
	}
	p.built = true
}

func (p *Collisions) useCollisions(handler CollisionHandler, hitDuration time.Duration) (*BroadPhase, *NarrowPhase) {
	return NewBroadPhase(p.space), NewNarrowPhase(p.space, handler, hitDuration)
}
