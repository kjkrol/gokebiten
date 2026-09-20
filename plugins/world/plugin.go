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
	"github.com/kjkrol/uid"
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
	entKinds *EntKindDict
	seeded   []Entry

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
	dict := newEntKindDict()
	m.entKinds = dict
	return &Plugin{Res: Resources{Config: cfg, Telemetry: &m.telemetry, Camera: cam}, module: m, entKinds: dict}
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
	ctx.Setup(p.entKinds) // joins the tracked resources, so the type dictionary is saved
	return nil
}

// RunPlan runs world's movement pipeline for this tick — call from your own Game.RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.module.RunPlan(ctx, d)
}

// WithRenderer builds this plugin's own entity renderer, drawing cam-relative sprites from atlas.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.renderer = newRenderer(p.Res.Camera, atlas, p.Res.Config.Space.Width, p.Res.Config.Space.Height)
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

// RegisterBehavior adds a world.Behavior to the decision pass world runs each
// tick before movement, in registration order — so one consuming what earlier
// ones decided sees it. Anything else is reported as ErrUnhostedBehavior.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	for _, b := range behaviors {
		system, ok := b.(Behavior)
		if !ok {
			return fmt.Errorf("%w: %T in %s", plugin.ErrUnhostedBehavior, b, p.Name())
		}
		p.module.RegisterBehavior(system)
	}
	return nil
}

// =================================================================
// world-specific
// =================================================================

// Seed adds entries to the entities spawned when this Stage starts fresh — see Populate.
func (p *Plugin) Seed(entries ...Entry) { p.seeded = append(p.seeded, entries...) }

// Populate spawns every seeded entity, erroring (and spawning nothing) on an unknown kind or Data a kind's templates reject.
func (p *Plugin) Populate() error {
	var order []string
	groups := make(map[string][]any)
	for _, e := range p.seeded {
		kind, ok := p.entKinds.Get(e.kind)
		if !ok {
			return fmt.Errorf("world: unknown EntKind %q", e.kind)
		}
		if err := kind.validate(e.data); err != nil {
			return err
		}
		if _, seen := groups[e.kind]; !seen {
			order = append(order, e.kind)
		}
		groups[e.kind] = append(groups[e.kind], e.data)
	}
	for _, name := range order {
		kind, _ := p.entKinds.Get(name)
		p.module.populate(kind, groups[name])
	}
	p.seeded = nil
	return nil
}

// Despawn takes an entity out of the world: out of the ECS at the end of the
// tick, and out of the shared spatial index, so nothing goes on seeing a ghost.
func (p *Plugin) Despawn(cb *goke.CmdBuf, id uid.UID64) { p.module.despawn(cb, id) }

// Space returns world's shared spatial index — every Populate entity is kept in sync with it.
func (p *Plugin) Space() *gokg.Space { return p.module.space }

// MaxStep is the furthest a single entity can move in one tick — the cap
// MoveSystem applies. Two entities closing head-on therefore shut at most
// 2*MaxStep of gap per tick, which is the reach a broad phase needs.
func (p *Plugin) MaxStep() float64 { return p.module.maxStep() }

// EntityRenderer returns the concrete entity renderer for further chaining (WithOverlay, WithModify, ...), or nil.
func (p *Plugin) EntityRenderer() *Renderer { return p.renderer }

// RegisterSpeedModifier adds m to the set VelocitySystem folds into every entity's Velocity.Value each tick.
func (p *Plugin) RegisterSpeedModifier(m SpeedModifier) { p.module.RegisterSpeedModifier(m) }

// EntKindDict returns this Plugin's registered set of EntKinds — call
// Define to register kinds, Entry to build Seed's roster entries.
func (p *Plugin) EntKindDict() *EntKindDict { return p.entKinds }
