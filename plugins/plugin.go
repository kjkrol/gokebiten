package plugins

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/resources"
)

// Plugin extends a Game: Install wires an ECS module, setup, renderers, and/or resources as one unit.
type Plugin interface {
	// Name uniquely identifies this plugin — UsePlugin rejects a duplicate.
	Name() string

	// Install runs once its dependencies are available — return an error from Require to be retried automatically.
	Install(ctx *GameCtx) error

	// RunPlan runs this plugin's per-tick work — call from your own Game.Loop closure, in whatever order you need.
	RunPlan(ctx goke.RunCtx, d time.Duration)

	// WithRenderer configures this plugin's own render.Renderer to draw cam-relative sprites from atlas — call before UsePlugin. A no-op for a plugin with no renderer of its own.
	WithRenderer(cam camera.Camera, atlas render.AtlasSource)

	// Renderer returns this plugin's own render.Renderer, or nil if it has none.
	Renderer() render.Renderer

	// EventHandler returns this plugin's own control.EventHandler, or nil if it has none.
	EventHandler() control.EventHandler

	// Resources returns this plugin's single published resource, or nil if it has none.
	Resources() resources.Resources
}
