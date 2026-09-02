package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires Collisions into a Game — optional, borrows world.Plugin's own
// Space. Must never import collisions/strategies/*; compose handlers via SetCollisionHandlers instead.
type Plugin struct {
	handlers   []CollisionHandler
	hitExpires time.Duration

	collisions *Collisions
}

var _ gokebiten.Plugin = (*Plugin)(nil)

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) SetCollisionHandlers(handlers ...CollisionHandler) *Plugin {
	p.handlers = handlers
	return p
}

// SetHitExpires enables Collisions.SetHitExpires with the given duration.
func (p *Plugin) SetHitExpires(d time.Duration) *Plugin {
	p.hitExpires = d
	return p
}

func (p *Plugin) Name() string { return "gokebiten.collisions" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	worldPlugin, err := ctx.Require[*world.Plugin]()
	if err != nil {
		return err
	}
	space := worldPlugin.World().Space()

	p.collisions = New(space, ctx.ECS())
	if len(p.handlers) > 0 {
		p.collisions.SetCollisionHandlers(p.handlers...)
	}
	if p.hitExpires > 0 {
		p.collisions.SetHitExpires(p.hitExpires)
	}

	ctx.UseModule(p.collisions)
	return nil
}

// RunPlan runs the collision engine for this tick — call from your own Game.Loop closure.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.collisions.RunPlan(ctx, d) }

// Renderer is a no-op — collisions has no render.Renderer of its own.
func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler is a no-op — collisions has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }
