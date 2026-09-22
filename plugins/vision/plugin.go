package vision

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

// Plugin wires vision into a Stage over world.Plugin's space and camera.
// It publishes what entities can see; what to do about it is a behavior's business.
type Plugin struct {
	worldPlugin *world.Plugin
	camera      camera.Camera
	module      *module
	renderer    *Renderer
	style       ConeStyle

	sightings plugin.PairHost[Sighting]
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds the vision plugin over worldPlugin's shared spatial index.
func NewPlugin(worldPlugin *world.Plugin) *Plugin {
	return &Plugin{worldPlugin: worldPlugin, camera: worldPlugin.Camera()}
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gram.vision" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	p.module = newModule(p.worldPlugin.Space(), &p.sightings)
	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs the scan for this tick — call from your own Game.Loop closure.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer builds the cone renderer; atlas is unused, vision draws primitives.
func (p *Plugin) WithRenderer(render.AtlasSource) {
	p.renderer = NewRenderer(p.camera, p.worldPlugin.Space())
	if p.style != nil {
		p.renderer.WithStyle(p.style)
	}
}

func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler is a no-op — vision reads no input.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

// Serializable returns nil: vision keeps no state beside its components.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// RegisterBehavior hosts a plugin.Between of Sighting, run once per observer; call before Use.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	for _, b := range behaviors {
		if err := p.sightings.Add(b); err != nil {
			return fmt.Errorf("%w in %s — it takes Between for Sighting", err, p.Name())
		}
	}
	return nil
}

// =================================================================
// vision-specific
// =================================================================

// WithStyle sets how cones are drawn, in place of DefaultConeStyle; call before Use.
func (p *Plugin) WithStyle(style ConeStyle) *Plugin {
	p.style = style
	return p
}
