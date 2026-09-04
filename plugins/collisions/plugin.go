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
	handlers   []CollisionHandler
	hitExpires time.Duration
	module     *module
}

var _ plugins.Plugin = (*Plugin)(nil)

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) SetCollisionHandlers(handlers ...CollisionHandler) *Plugin {
	p.handlers = handlers
	return p
}

// SetHitExpires sets how long a Hit tag lingers before auto-expiring.
func (p *Plugin) SetHitExpires(d time.Duration) *Plugin {
	p.hitExpires = d
	return p
}

func (p *Plugin) Name() string { return "gokebiten.collisions" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	worldPlugin, err := ctx.Require[*world.Plugin]()
	if err != nil {
		return err
	}
	space := worldPlugin.Space()

	p.module = New(space, ctx.ECS())
	if len(p.handlers) > 0 {
		p.module.SetCollisionHandlers(p.handlers...)
	}
	if p.hitExpires > 0 {
		p.module.SetHitExpires(p.hitExpires)
	}

	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs the collision engine for this tick — call from your own Game.Loop closure.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// Renderer is a no-op — collisions has no render.Renderer of its own.
func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler is a no-op — collisions has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }
