package selection

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/render"
)

// Plugin wires selection into a Game; it depends on world and its Camera.
type Plugin struct {
	state       *Resources
	worldPlugin *world.Plugin
	camera      camera.Camera
	module      *module
	renderer    *Renderer
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds the selection plugin over worldPlugin's space and camera.
func NewPlugin(worldPlugin *world.Plugin) *Plugin {
	return &Plugin{state: &Resources{}, worldPlugin: worldPlugin, camera: worldPlugin.Camera()}
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gram.selection" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	sys := NewSelectionSystem(p.state, p.worldPlugin.Space(), p.camera)
	p.module = &module{sys: sys}
	ctx.UseModule(p.module)
	return nil
}

func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer builds the highlight renderer; atlas is unused, selection draws primitives.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.renderer = NewRenderer(p.camera, p.state)
}

func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler returns the default left-click and drag handler for selection.
func (p *Plugin) EventHandler() control.EventHandler { return NewDefaultEventHandler(p.state) }

// Serializable is a no-op — selection has nothing to persist.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// RegisterBehavior reports ErrUnhostedBehavior — selection hosts no behaviors.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	for _, b := range behaviors {
		return fmt.Errorf("%w: %T in %s", plugin.ErrUnhostedBehavior, b, p.Name())
	}
	return nil
}
