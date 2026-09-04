package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires selection into a Game — depends on:
//   - world (shared spatial index)
//   - registered camera plugin (screen<->world conversion)
type Plugin struct {
	state    *State
	sys      *SelectionSystem
	module   *module
	renderer *Renderer
}

var _ plugins.Plugin = (*Plugin)(nil)

func NewPlugin() *Plugin { return &Plugin{state: &State{}} }

// =================================================================
// plugins.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.selection" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	worldPlugin, err := ctx.Require[*world.Plugin]()
	if err != nil {
		return err
	}
	camera, err := ctx.Require[render.Camera]()
	if err != nil {
		return err
	}
	p.sys = NewSelectionSystem(p.state, worldPlugin.Space(), camera)
	p.module = &module{sys: p.sys}
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
