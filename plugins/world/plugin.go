package world

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg"
)

// Resources is world's single published Resources.
type Resources struct {
	Config    Config
	Telemetry *Telemetry
}

// Plugin builds a world - the mandatory foundation for any game with
// moving, drawable entities - and publishes Resources as a resource.
type Plugin struct {
	Res      Resources
	module   *module
	renderer *Renderer
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds Plugin around a fresh world — Populate/Space are usable
// immediately, before Install (e.g. in tests).
func NewPlugin(cfg Config) *Plugin {
	m := newModule(cfg)
	return &Plugin{Res: Resources{Config: cfg, Telemetry: &m.telemetry}, module: m}
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.world" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs world's movement pipeline for this tick — call from your own Game.RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.module.RunPlan(ctx, d)
}

// WithRenderer builds this plugin's own entity renderer, drawing cam-relative sprites from atlas.
func (p *Plugin) WithRenderer(cam camera.Camera, atlas render.AtlasSource) {
	p.renderer = newRenderer(cam, atlas)
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler is a no-op — world has no control.EventHandler of its own.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

// Serializable is a no-op — world has nothing to persist.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// =================================================================
// world-specific
// =================================================================

// Populate queues a spawn of count entities — see ExamplePlugin_Populate.
func (p *Plugin) Populate(count int, spawner *Spawner) *Plugin {
	p.module.Populate(count, spawner)
	return p
}

// Space returns world's shared spatial index — every Populate entity is kept in sync with it.
func (p *Plugin) Space() *gokg.Space { return p.module.space }

// EntityRenderer returns the concrete entity renderer for further chaining (WithOverlay, WithModify, ...), or nil.
func (p *Plugin) EntityRenderer() *Renderer { return p.renderer }

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (p *Plugin) RegisterSpeedModifier(m SpeedModifier) { p.module.RegisterSpeedModifier(m) }
