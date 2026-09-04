package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/uid"
)

// Plugin wires selection into a Game — depends on world (shared spatial index) and a registered camera plugin (screen<->world conversion).
type Plugin struct {
	state    *State
	module   *module
	renderer *Renderer
}

var _ plugins.Plugin = (*Plugin)(nil)

func NewPlugin() *Plugin {
	state := &State{}
	return &Plugin{state: state, module: NewSystem(state)}
}

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
	p.module.bindSpace(worldPlugin.Space())
	p.module.bindCamera(camera)
	ctx.Provide(p.state)
	ctx.UseModule(p.module)
	return nil
}

// WithRenderer builds this plugin's own highlight renderer (outline for every Selected entity, plus the drag marquee).
func (p *Plugin) WithRenderer() *Plugin {
	p.renderer = NewRenderer(p.state)
	return p
}

func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler returns the default left-click/drag control.EventHandler for selection — write your own against State for a different binding scheme.
func (p *Plugin) EventHandler() control.EventHandler { return NewDefaultEventHandler(p.state) }

func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// Select replaces the current selection with exactly ids — for programmatic
// selection (e.g. tagging entities Selected at spawn), independent of mouse input.
func (p *Plugin) Select(ids []uid.UID64) { p.module.Select(ids) }
