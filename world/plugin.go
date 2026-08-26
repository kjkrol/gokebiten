package world

import (
	"github.com/kjkrol/gokebiten"
)

// Plugin builds a Module and publishes Config, its *gokg.Space, and its *Telemetry as resources.
// Deliberately does not publish *Module itself — see Resources' doc comment.
type Plugin struct {
	config     Config
	population Population

	world *Module
}

var _ gokebiten.Plugin = (*Plugin)(nil)
var _ gokebiten.PostLoader = (*Module)(nil)

func NewPlugin(cfg Config, pop Population) *Plugin {
	return &Plugin{config: cfg, population: pop}
}

func (p *Plugin) Name() string { return "gokebiten.world" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	p.world = NewModule(p.config, p.population)
	ctx.Setup(p.world)
	ctx.Provide(p.config)
	ctx.Provide(p.world.Space())
	ctx.Provide(p.world.Telemetry())
	return nil
}

// World returns the underlying Module, built during Install — nil before that.
func (p *Plugin) World() *Module { return p.world }
