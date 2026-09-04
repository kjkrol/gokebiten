package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires selection into a Game — depends on a registered camera plugin (screen<->world conversion).
type Plugin struct {
	state       *State
	worldPlugin *world.Plugin
	module      *module
	renderer    *Renderer
}

var _ plugins.Plugin = (*Plugin)(nil)

// NewPlugin builds the selection plugin over worldPlugin's shared spatial index.
func NewPlugin(worldPlugin *world.Plugin) *Plugin {
	return &Plugin{state: &State{}, worldPlugin: worldPlugin}
}

// =================================================================
// plugins.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.selection" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	// Ordering guard, not a data dependency: waits until world's own Install
	// (and its ctx.UseModule) has actually run, so SelectionSystem's query
	// builds after world's entities exist in the same ecs.Setup.
	if err := ctx.RequirePlugin(p.worldPlugin); err != nil {
		return err
	}
	camera, err := ctx.Require[render.Camera]()
	if err != nil {
		return err
	}
	sys := NewSelectionSystem(p.state, p.worldPlugin.Space(), camera)
	p.module = &module{sys: sys}
	ctx.Provide(p.state)
	ctx.UseModule(p.module)
	return nil
}

func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer builds this plugin's own highlight renderer (outline for every Selected
// entity,plus the drag marquee) — atlas is unused, selection draws primitives.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.renderer = NewRenderer(p.state)
}

func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler returns the default left-click/drag control.EventHandler for selection
// or write your own against State for a different binding scheme.
func (p *Plugin) EventHandler() control.EventHandler { return NewDefaultEventHandler(p.state) }
