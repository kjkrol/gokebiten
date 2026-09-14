package world

import (
	"fmt"
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
	Camera    camera.Camera
}

var _ plugin.Serializable = (*Resources)(nil)

// Persisted returns the camera's Viewport/Zoom for Persistence.Save/Load to include automatically.
func (r *Resources) Persisted() []any { return r.Camera.Persisted() }

// Plugin is the world a Stage installs via ctx.UseWorld — never construct
// and Use your own.
type Plugin struct {
	Res      Resources
	module   *module
	renderer *Renderer
	entKinds *entKindDict
	seeded   Roster

	cameraControls bool
	scrollSpeed    int32
}

var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.Builtin = (*Plugin)(nil)
var _ plugin.Restorer = (*Plugin)(nil)
var _ plugin.Populator = (*Plugin)(nil)

// Builtin marks Plugin as installed by the engine itself — see ctx.UseWorld.
func (*Plugin) Builtin() {}

// NewPlugin builds Plugin around a fresh world — Seed/Populate/Space are
// usable immediately, before Install (e.g. in tests).
func NewPlugin(cfg Config) *Plugin {
	m := newModule(cfg)
	cam := camera.NewFromSpaceWithConfig(cfg.Space.Width, cfg.Space.Height, cfg.Space.Toroidal, cfg.Camera)
	return &Plugin{Res: Resources{Config: cfg, Telemetry: &m.telemetry, Camera: cam}, module: m, entKinds: newEntKindDict()}
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

// Seed adds roster to the entities spawned when this Stage starts fresh — see Populate.
func (p *Plugin) Seed(roster Roster) { p.seeded = append(p.seeded, roster...) }

// Populate spawns every seeded entity, erroring (and spawning nothing) on an unknown kind or Data a kind's templates reject.
func (p *Plugin) Populate() error {
	var order []string
	groups := make(map[string][]any)
	for _, e := range p.seeded {
		kind, ok := p.entKinds.Get(e.Kind)
		if !ok {
			return fmt.Errorf("world: unknown EntKind %q", e.Kind)
		}
		if err := kind.validate(e.Data); err != nil {
			return err
		}
		if _, seen := groups[e.Kind]; !seen {
			order = append(order, e.Kind)
		}
		groups[e.Kind] = append(groups[e.Kind], e.Data)
	}
	for _, name := range order {
		kind, _ := p.entKinds.Get(name)
		p.module.populate(kind, groups[name])
	}
	p.seeded = nil
	return nil
}

// Space returns world's shared spatial index — every Populate entity is kept in sync with it.
func (p *Plugin) Space() *gokg.Space { return p.module.space }

// EntityRenderer returns the concrete entity renderer for further chaining (WithOverlay, WithModify, ...), or nil.
func (p *Plugin) EntityRenderer() *Renderer { return p.renderer }

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (p *Plugin) RegisterSpeedModifier(m SpeedModifier) { p.module.RegisterSpeedModifier(m) }

// EntKindDict returns this Plugin's registered set of EntKinds — call
// Create to register kinds, Get/All to read them back.
func (p *Plugin) EntKindDict() EntKindDict { return p.entKinds }
