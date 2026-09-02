package world

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin builds a Module — the mandatory foundation for any game with
// moving, drawable entities — and publishes Config as a resource.
type Plugin struct {
	config Config

	world    *Module
	renderer *Renderer
}

var _ gokebiten.Plugin = (*Plugin)(nil)
var _ gokebiten.PostLoader = (*Module)(nil)

func NewPlugin(cfg Config) *Plugin {
	return &Plugin{config: cfg}
}

func (p *Plugin) Name() string { return "gokebiten.world" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	if p.renderer != nil {
		if _, err := ctx.Require[render.Camera](); err != nil {
			return err
		}
	}
	p.world = NewModule(p.config)
	p.world.step = ctx.Step()
	p.world.ecs = ctx.ECS()
	ctx.Setup(p.world)
	ctx.Provide(p.config)
	ctx.Provide(p.world.Telemetry())
	ctx.Provide(p)
	return nil
}

// World returns the underlying Module, built during Install — nil before that.
func (p *Plugin) World() *Module { return p.world }

// WithRenderer builds this plugin's own entity renderer, drawing sprites from atlas.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) *Plugin {
	p.renderer = newRenderer(atlas)
	return p
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EntityRenderer returns the concrete entity renderer for further chaining (WithOverlay, WithModify, ...), or nil.
func (p *Plugin) EntityRenderer() *Renderer { return p.renderer }

// EventHandler is a no-op — world has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (p *Plugin) RegisterSpeedModifier(m SpeedModifier) { p.world.RegisterSpeedModifier(m) }

// RunPlan runs world's movement pipeline for this tick — call from your own Game.Loop closure.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.world.RunPlan(ctx, d)
}
