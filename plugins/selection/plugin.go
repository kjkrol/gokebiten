package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires System into a Game — depends on world (shared spatial index) and a registered camera plugin (screen<->world conversion).
type Plugin struct {
	state  *State
	system *System
}

var _ gokebiten.Plugin = (*Plugin)(nil)

func NewPlugin() *Plugin {
	state := &State{}
	return &Plugin{state: state, system: NewSystem(state)}
}

func (p *Plugin) Name() string { return "gokebiten.selection" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	worldPlugin, err := ctx.Require[*world.Plugin]()
	if err != nil {
		return err
	}
	camera, err := ctx.Require[render.Camera]()
	if err != nil {
		return err
	}
	p.system.bindSpace(worldPlugin.World().Space())
	p.system.bindCamera(camera)
	ctx.Provide(p.state)
	ctx.UseModule(p.system)
	return nil
}

// System returns the underlying System.
func (p *Plugin) System() *System { return p.system }

// State returns the published input state — write to it yourself for a custom EventHandler, or read EventHandler for the default one.
func (p *Plugin) State() *State { return p.state }

func (p *Plugin) Renderer() render.Renderer { return nil }

// EventHandler returns the default left-click/drag control.EventHandler for selection — write your own against State for a different binding scheme.
func (p *Plugin) EventHandler() control.EventHandler { return NewDefaultEventHandler(p.state) }

func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.system.RunPlan(ctx, d) }
