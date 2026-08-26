package physics

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokg"
)

// Plugin wires Physics into a Game; requires a *gokg.Space resource (e.g. from world.Plugin) registered earlier.
type Plugin struct {
	minEntitySize uint32
	handlers      []collisions.CollisionHandler
	debug         bool
	hitExpires    time.Duration
	extraSystems  []goke.System

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

func (p *Plugin) Name() string { return "gokebiten.physics" }

func (p *Plugin) Install(ctx *gokebiten.PluginContext) error {
	space, ok := ctx.Resources.TryGetResource[*gokg.Space]()
	if !ok {
		return fmt.Errorf("physics: requires a *gokg.Space resource — install a world plugin (e.g. world.Plugin) before physics.Plugin")
	}

	p.physics = New(space, ctx.ECS(), p.minEntitySize, ctx.Step())
	if len(p.handlers) > 0 {
		p.physics.SetCollisionHandlers(p.handlers...)
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
