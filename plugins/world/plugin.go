package world

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg"
)

// Plugin builds a world — the mandatory foundation for any game with
// moving, drawable entities — and publishes Config as a resource.
type Plugin struct {
	config   Config
	module   *module
	renderer *Renderer
}

var _ plugins.Plugin = (*Plugin)(nil)

// NewPlugin builds Plugin around a fresh world — Populate/Space are usable
// immediately, before Install (e.g. in tests).
func NewPlugin(cfg Config) *Plugin {
	return &Plugin{config: cfg, module: newModule(cfg)}
}

func (p *Plugin) Name() string { return "gokebiten.world" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	if p.renderer != nil {
		if _, err := ctx.Require[render.Camera](); err != nil {
			return err
		}
	}
	p.module.step = ctx.Step()
	ctx.UseModule(p.module)
	ctx.Provide(p.config)
	ctx.Provide(p.module.Telemetry())
	ctx.Provide(p)
	return nil
}

// ModuleHandle is the goke.Module/gokebiten.PostLoader surface Plugin
// registers on world's behalf — returned by Plugin.Module for advanced use
// (e.g. composing a focused test's own ecs.Setup call).
type ModuleHandle interface {
	goke.Module
	gokebiten.PostLoader
}

// Module returns the underlying ModuleHandle — an escape hatch for advanced
// use; most callers want Populate/Space/RegisterSpeedModifier instead.
func (p *Plugin) Module() ModuleHandle { return p.module }

// Populate queues a spawn of count entities — see ExamplePlugin_Populate.
func (p *Plugin) Populate(count int, spawner *Spawner) *Plugin {
	p.module.Populate(count, spawner)
	return p
}

// Space returns world's shared spatial index — every Populate entity is kept in sync with it.
func (p *Plugin) Space() *gokg.Space { return p.module.Space() }

// Telemetry returns the entity-count telemetry Populate/PostLoad maintain.
func (p *Plugin) Telemetry() *Telemetry { return p.module.Telemetry() }

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
func (p *Plugin) RegisterSpeedModifier(m SpeedModifier) { p.module.RegisterSpeedModifier(m) }

// RunPlan runs world's movement pipeline for this tick — call from your own Game.Loop closure.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.module.RunPlan(ctx, d)
}
