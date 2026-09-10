package selection

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/resources"
)

// Plugin wires selection into a Game — depends on worldPlugin and a Camera for screen<->world conversion.
type Plugin struct {
	state       *Resources
	worldPlugin *world.Plugin
	camera      camera.Camera
	module      *module
	renderer    *Renderer
}

var _ plugins.Plugin = (*Plugin)(nil)

// NewPlugin builds the selection plugin over worldPlugin's shared spatial index, using cam for click/drag hit-testing.
func NewPlugin(worldPlugin *world.Plugin, cam camera.Camera) *Plugin {
	return &Plugin{state: &Resources{}, worldPlugin: worldPlugin, camera: cam}
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
	sys := NewSelectionSystem(p.state, p.worldPlugin.Space(), p.camera)
	p.module = &module{sys: sys}
	ctx.UseModule(p.module)
	return nil
}

func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer builds this plugin's own highlight renderer (outline for every Selected
// entity,plus the drag marquee) — atlas is unused, selection draws primitives.
func (p *Plugin) WithRenderer(cam camera.Camera, atlas render.AtlasSource) {
	p.renderer = NewRenderer(cam, p.state)
}

func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler returns the default left-click/drag control.EventHandler for selection
// or write your own against Resources for a different binding scheme.
func (p *Plugin) EventHandler() control.EventHandler { return NewDefaultEventHandler(p.state) }

// Resources returns selection's single published Resources.
func (p *Plugin) Resources() resources.Resources { return p.state }
