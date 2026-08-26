package world

import (
	"github.com/kjkrol/gokebiten"
)

// Plugin builds a Module and publishes Config and its *gokg.Space as resources.
// Deliberately does not publish *Module itself — see Resources' doc comment.
type Plugin struct {
	config      Config
	population  Population
	onReindexed func(count int)

	world *Module
}

var _ gokebiten.Plugin = (*Plugin)(nil)
var _ gokebiten.PostLoader = (*Module)(nil)

func NewPlugin(cfg Config, pop Population) *Plugin {
	return &Plugin{config: cfg, population: pop}
}

// OnReindexed sets the callback Module.PostLoad invokes with the count of reindexed entities.
func (p *Plugin) OnReindexed(fn func(count int)) *Plugin {
	p.onReindexed = fn
	return p
}

func (p *Plugin) Name() string { return "gokebiten.world" }

func (p *Plugin) Install(ctx *gokebiten.PluginContext) error {
	p.world = NewModule(p.config, p.population)
	p.world.onReindexed = p.onReindexed
	ctx.Setup(p.world)
	ctx.Resources.InsertResource(p.config)
	ctx.Resources.InsertResource(p.world.Space())
	return nil
}

// World returns the underlying Module, built during Install — nil before that.
func (p *Plugin) World() *Module { return p.world }
