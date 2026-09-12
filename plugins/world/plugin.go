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

// Batch pairs a spawn count with the Spawner producing its entities — see game.Game.Spawn.
type Batch struct {
	Count   int
	Spawner *Spawner
}

// Resources is world's single published Resources.
type Resources struct {
	Config    Config
	Telemetry *Telemetry
	Camera    camera.Camera
}

var _ plugin.Serializable = (*Resources)(nil)

// Persisted returns the camera's Viewport/Zoom for Persistence.Save/Load to include automatically.
func (r *Resources) Persisted() []any { return r.Camera.Persisted() }

// Plugin builds a world - the mandatory foundation for any game with
// moving, drawable entities - and publishes Resources as a resource.
// Engine installs this automatically — do not construct/Use your own; get
// the running instance via ctx.World().
type Plugin struct {
	Res      Resources
	module   *module
	renderer *Renderer

	cameraControls bool
	scrollSpeed    int32
}

var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.Builtin = (*Plugin)(nil)
var _ plugin.Restorer = (*Plugin)(nil)

// Builtin marks Plugin as installed automatically by Engine — see ctx.World().
func (*Plugin) Builtin() {}

// NewPlugin builds Plugin around a fresh world — Populate/Space are usable
// immediately, before Install (e.g. in tests).
func NewPlugin(cfg Config) *Plugin {
	m := newModule(cfg)
	cam := camera.NewFromSpaceWithConfig(cfg.Space.Width, cfg.Space.Height, cfg.Space.Toroidal, cfg.Camera)
	return &Plugin{Res: Resources{Config: cfg, Telemetry: &m.telemetry, Camera: cam}, module: m}
}

// WithCameraControls enables the default wheel-zoom/middle-drag-pan/edge-scroll EventHandler.
func (p *Plugin) WithCameraControls(scrollSpeed ...int32) *Plugin {
	p.cameraControls = true
	p.scrollSpeed = defaultCameraScrollSpeed
	if len(scrollSpeed) > 0 {
		p.scrollSpeed = scrollSpeed[0]
	}
	return p
}

// Camera returns world's shared Camera, built from Config.Space.
func (p *Plugin) Camera() camera.Camera { return p.Res.Camera }

// Restore applies the camera's Viewport/Zoom decoded by Persistence.Load.
func (p *Plugin) Restore() { p.Res.Camera.Restore() }

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
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.renderer = newRenderer(p.Res.Camera, atlas)
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler returns the default wheel-zoom/middle-drag-pan/edge-scroll
// handler, or nil unless WithCameraControls was called.
func (p *Plugin) EventHandler() control.EventHandler {
	if !p.cameraControls {
		return nil
	}
	return newDefaultCameraHandler(p.Res.Camera, p.scrollSpeed)
}

// Serializable returns world's persistable state (its camera's Viewport/Zoom).
func (p *Plugin) Serializable() plugin.Serializable { return &p.Res }

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
