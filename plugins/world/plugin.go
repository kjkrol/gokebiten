package world

import (
	"fmt"
	"reflect"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world/kind"
	"github.com/kjkrol/gokebiten/render"
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
	kinds    *Kinds
	seeded   []kind.Entry

	cameraControls bool
	scrollSpeed    int32
}

var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.Builtin = (*Plugin)(nil)
var _ plugin.Restorer = (*Plugin)(nil)
var _ plugin.Populator = (*Plugin)(nil)

// Builtin marks Plugin as installed by the engine itself — see ctx.UseWorld.
func (*Plugin) Builtin() {}

// NewPlugin builds Plugin around a fresh world, usable before Install.
func NewPlugin(cfg Config) *Plugin {
	m := newModule(cfg)
	cam := camera.NewFromSpaceWithConfig(cfg.Space.Width, cfg.Space.Height, cfg.Space.Edges, cfg.Camera)
	kinds := newKinds()
	m.kinds = kinds
	return &Plugin{Res: Resources{Config: cfg, Telemetry: &m.telemetry, Camera: cam}, module: m, kinds: kinds}
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
	p.module.ecs = ctx.ECS()
	ctx.UseModule(p.module)
	ctx.Setup(p.kinds)
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

// EventHandler returns the zoom, pan and edge-scroll handler, or nil without WithCameraControls.
func (p *Plugin) EventHandler() control.EventHandler {
	if !p.cameraControls {
		return nil
	}
	return newDefaultCameraHandler(p.Res.Camera, p.scrollSpeed)
}

// Serializable returns world's persistable state (its camera's Viewport/Zoom).
func (p *Plugin) Serializable() plugin.Serializable { return &p.Res }

// RegisterBehavior adds world.Behaviors to the decision pass run before movement, in order.
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
func (p *Plugin) Seed(entries ...kind.Entry) { p.seeded = append(p.seeded, entries...) }

// Populate spawns every seeded entity, or none and an error on an unknown kind or a wrong row.
func (p *Plugin) Populate() error {
	var order []string
	groups := make(map[string][]any)
	for _, e := range p.seeded {
		r, ok := p.kinds.entries[e.Kind()]
		if !ok {
			return fmt.Errorf("world: unknown kind %q", e.Kind())
		}
		if got := reflect.TypeOf(e.Row()); got != r.row {
			return fmt.Errorf("world: kind %q: an entry carries a %v, its rows are %v", r.name, got, r.row)
		}
		if _, seen := groups[r.name]; !seen {
			order = append(order, r.name)
		}
		groups[r.name] = append(groups[r.name], e.Row())
	}
	for _, name := range order {
		p.module.populate(p.kinds.entries[name], groups[name])
	}
	p.seeded = nil
	return nil
}

// Attach gives id the component v from the next sync on, replacing one it already carries.
func (p *Plugin) Attach[T any](cb *goke.CmdBuf, id uid.UID64, v T) {
	cb.AddOne(id, compID[T](p), v)
}

// Detach takes T off id from the next sync on; an entity without it is left alone.
func (p *Plugin) Detach[T any](cb *goke.CmdBuf, id uid.UID64) {
	if reflect.TypeFor[T]() == reflect.TypeFor[Base]() {
		panic("world: Detach[Base] — every entity carries a Base; Despawn the entity instead")
	}
	cb.RemoveCompOne(id, compID[T](p))
}

// Declare tells save files about T, a component only ever attached; call it in Stage.Init.
func (p *Plugin) Declare[T any]() {
	p.module.declared = append(p.module.declared, goke.LoadComp[T]())
}

func compID[T any](p *Plugin) goke.CompID {
	if p.module.ecs == nil {
		panic("world: Attach/Detach before the Plugin was installed")
	}
	return p.module.ecs.RegComp[T]()
}

// Despawn takes an entity out of the ECS at the end of the tick.
func (p *Plugin) Despawn(cb *goke.CmdBuf, id uid.UID64) { p.module.despawn(cb, id) }

// OnExit sets what happens, once, to an entity leaving by an open edge; unset, it is despawned.
func (p *Plugin) OnExit(fn func(t plugin.Tick, id uid.UID64)) { p.module.exits.onExit = fn }

// Tracked takes what the space said of a move a sibling plugin made for id.
func (p *Plugin) Tracked(t plugin.Tick, id uid.UID64, inside bool) { p.module.tracked(t, id, inside) }

// Space returns world's shared space, rebuilt from every entity each tick after movement.
func (p *Plugin) Space() *aabbworld.Space { return p.module.space }

// EntityRenderer returns the entity renderer for further chaining, or nil.
func (p *Plugin) EntityRenderer() *Renderer { return p.renderer }

// RegisterSpeedModifier adds m to the factors folded into every entity's speed each tick.
func (p *Plugin) RegisterSpeedModifier(m SpeedModifier) { p.module.RegisterSpeedModifier(m) }

// Kinds returns this Plugin's registry of entity kinds — what kind.Define registers with.
func (p *Plugin) Kinds() *Kinds { return p.kinds }
