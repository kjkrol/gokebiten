package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires the collision engine into a Game — optional, borrows world.Plugin's own Space.
// Must never import collisions/strategies/*; a game registers those with RegisterBehavior.
type Plugin struct {
	worldPlugin *world.Plugin
	module      *module

	pairs    plugin.PairHost[Meeting]
	entities plugin.EachHost[Struck]
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds the collisions plugin over worldPlugin's shared spatial index.
func NewPlugin(worldPlugin *world.Plugin) *Plugin {
	return &Plugin{worldPlugin: worldPlugin}
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.collisions" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	space := p.worldPlugin.Space()

	// Two entities closing head-on shut 2*MaxStep of gap per tick, so that is
	// exactly how far the broad phase has to see — taken from world rather
	// than guessed, so it follows the entity size instead of contradicting it.
	p.module = newModule(space, ctx.ECS(), 2*p.worldPlugin.MaxStep(), &p.pairs, &p.entities)
	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs the collision engine for this tick — call from your own Game.Loop closure.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer is a no-op — collisions has no render.Renderer of its own.
func (p *Plugin) WithRenderer(render.AtlasSource) {}

// Renderer is a no-op — collisions has no render.Renderer of its own.
func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler is a no-op — collisions has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

// Serializable is a no-op — collisions has nothing to persist.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// RegisterBehavior hosts a plugin.Between behavior made for Meeting in the narrow
// phase, or a plugin.Each one made for Struck in the broad phase — before Use.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	return host(&p.pairs, &p.entities, behaviors)
}
