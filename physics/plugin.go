package physics

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/stats"
	"github.com/kjkrol/gokg"
)

// Plugin wires Physics into a Game; needs a *gokg.Space resource, published by any installed world plugin.
type Plugin struct {
	minEntitySize uint32
	handlers      []collisions.CollisionHandler
	debug         bool
	hitExpires    time.Duration
	extraSystems  []goke.System
	stats         *stats.Stats

	physics *Physics
}

var _ gokebiten.Plugin = (*Plugin)(nil)

// NewPlugin builds a Plugin for the given minimum entity size — see New.
func NewPlugin(minEntitySize uint32) *Plugin {
	return &Plugin{minEntitySize: minEntitySize}
}

func (p *Plugin) SetCollisionHandlers(handlers ...collisions.CollisionHandler) *Plugin {
	p.handlers = handlers
	return p
}

func (p *Plugin) DebugEnabled() *Plugin {
	p.debug = true
	return p
}

// SetHitExpires enables Physics.SetHitExpires with the given duration.
func (p *Plugin) SetHitExpires(d time.Duration) *Plugin {
	p.hitExpires = d
	return p
}

// RegSys registers an extra system via Physics.RegSys, in call order.
func (p *Plugin) RegSys(sys goke.System) *Plugin {
	p.extraSystems = append(p.extraSystems, sys)
	return p
}

// EnableStats wires a collision counter into the handler chain and publishes it as a *stats.Stats resource.
func (p *Plugin) EnableStats() *Plugin {
	p.stats = &stats.Stats{}
	return p
}

func (p *Plugin) Name() string { return "gokebiten.physics" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	space, err := ctx.Require[*gokg.Space]()
	if err != nil {
		return err
	}

	handlers := p.handlers
	if p.stats != nil {
		handlers = append(handlers, stats.NewHandler(p.stats))
		ctx.Provide(p.stats)
	}

	p.physics = New(space, ctx.ECS(), p.minEntitySize, ctx.Step())
	if len(handlers) > 0 {
		p.physics.SetCollisionHandlers(handlers...)
	}
	if p.debug {
		p.physics.DebugEnabled()
	}
	if p.hitExpires > 0 {
		p.physics.SetHitExpires(p.hitExpires)
	}
	for _, sys := range p.extraSystems {
		p.physics.RegSys(sys)
	}

	ctx.UseModule(p.physics)
	return nil
}

// Physics returns the underlying Physics module, built during Install — nil before that.
func (p *Plugin) Physics() *Physics { return p.physics }

// Stats returns the collision counter published during Install, or nil if EnableStats wasn't called.
func (p *Plugin) Stats() *stats.Stats { return p.stats }
