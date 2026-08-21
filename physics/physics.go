package physics

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/debug"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokg"
)

var _ goke.Module = (*Physics)(nil)

type Physics struct {
	space         *gokg.Space
	ecs           *goke.ECS
	minEntitySize uint32
	physicsStep   time.Duration

	handlers     []collisions.CollisionHandler
	extra        []goke.System
	extraHandles []goke.Runnable
	debug        bool
	hitDuration  time.Duration

	moveSystem  goke.Runnable
	broadPhase  goke.Runnable
	narrowPhase goke.Runnable
	built       bool
}

// New builds Physics for the given world — minEntitySize is the smallest
// entity size the world was provisioned for, used to cap per-tick
// displacement so nothing can tunnel through it undetected. All systems
// (kinematics included) register lazily, on the first RunPlan/RegSystems,
// not here — so New itself never touches ecs, keeping it safe to call
// before ECS.Load.
func New(space *gokg.Space, ecs *goke.ECS, minEntitySize uint32, physicsStep time.Duration) *Physics {
	return &Physics{space: space, ecs: ecs, minEntitySize: minEntitySize, physicsStep: physicsStep}
}

func (p *Physics) SetCollisionHandlers(handlers ...collisions.CollisionHandler) *Physics {
	p.handlers = handlers
	return p
}

func (p *Physics) DebugEnabled() *Physics {
	p.debug = true
	return p
}

func (p *Physics) SetHitExpires(duration time.Duration) *Physics {
	p.hitDuration = duration
	p.extra = append(p.extra, NewTagExpirySystem(func(h *collisions.Hit) time.Time { return h.ExpiresAt() }))
	return p
}

func (p *Physics) RegSys(sys goke.System) *Physics {
	p.extra = append(p.extra, sys)
	return p
}

// RegSystems builds and registers the collision systems — see [goke.Module].
// NarrowPhase.Init scans existing entities for Sensor tags, so this must run
// after the world is populated, not before — RunPlan already calls it lazily
// at the right time; call this directly only if that ordering is guaranteed
// some other way.
func (p *Physics) RegSystems(ecs *goke.ECS) {
	if !p.built {
		p.build()
	}
}

func (p *Physics) RunPlan(ctx goke.RunCtx, d time.Duration) {
	if !p.built {
		p.build()
	}
	ctx.Run(p.moveSystem, d)
	ctx.Sync()

	ctx.Run(p.broadPhase, d)
	ctx.Sync()

	ctx.Run(p.narrowPhase, d)
	ctx.Sync()

	for _, h := range p.extraHandles {
		ctx.Run(h, d)
		ctx.Sync()
	}
}

// SetupSystems is empty — Physics has no one-time seeding of its own — but
// is required to satisfy goke.Module, which now embeds SetupProvider.
func (p *Physics) SetupSystems() []goke.System { return nil }

// LoadComps lists the component types kinematics and collisions own — see
// [goke.CompProvider].
func (p *Physics) LoadComps() []goke.CompToken {
	return []goke.CompToken{
		goke.LoadComp[kinematics.Position](),
		goke.LoadComp[kinematics.Velocity](),
		goke.LoadComp[collisions.Collision](),
		goke.LoadComp[collisions.Hit](),
		goke.LoadComp[collisions.Sensor](),
	}
}

func (p *Physics) build() {
	p.moveSystem = p.useKinematics()

	handlers := p.handlers
	if p.debug {
		handlers = append(handlers, debug.NewHandler())
	}
	broad, narrow := p.useCollisions(collisions.MultiHandler(handlers...), p.hitDuration)
	p.broadPhase = p.ecs.RegSys(broad)
	p.narrowPhase = p.ecs.RegSys(narrow)

	for _, sys := range p.extra {
		p.extraHandles = append(p.extraHandles, p.ecs.RegSys(sys))
	}
	p.built = true
}

func (p *Physics) useKinematics() goke.Runnable {
	// Cap displacement per tick to half the smallest entity's size, so
	// nothing can tunnel through it undetected.
	maxSpeed := int32(float64(p.minEntitySize) / 2 / p.physicsStep.Seconds())
	system := kinematics.NewSystem(p.space, maxSpeed)
	return p.ecs.RegSys(system)
}

func (p *Physics) useCollisions(handler collisions.CollisionHandler, hitDuration time.Duration) (*collisions.BroadPhase, *collisions.NarrowPhase) {
	return collisions.NewBroadPhase(p.space), collisions.NewNarrowPhase(p.space, handler, hitDuration)
}
