package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires the collision engine into a Game — optional, borrows world.Plugin's own Space.
// Must never import collisions/strategies/*; compose handlers via SetCollisionHandlers instead.
type Plugin struct {
	handlers    []CollisionHandler
	hitExpires  time.Duration
	worldPlugin *world.Plugin
	module      *module
}

var _ plugins.Plugin = (*Plugin)(nil)

// NewPlugin builds the collisions plugin over worldPlugin's shared spatial
// index — hitExpires is the default Hit lifetime for entities that don't
// supply their own HitExpires (0 for a Hit that never auto-expires on its own).
func NewPlugin(hitExpires time.Duration, worldPlugin *world.Plugin) *Plugin {
	return &Plugin{hitExpires: hitExpires, worldPlugin: worldPlugin}
}

// =================================================================
// plugins.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.collisions" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	if err := ctx.RequirePlugin(p.worldPlugin); err != nil {
		return err
	}
	space := p.worldPlugin.Space()

	p.module = New(space, ctx.ECS(), p.hitExpires)
	if len(p.handlers) > 0 {
		p.module.SetCollisionHandlers(p.handlers...)
	}

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

// =================================================================
// collisions-specific
// =================================================================

func (p *Plugin) SetCollisionHandlers(handlers ...CollisionHandler) *Plugin {
	p.handlers = handlers
	return p
}
