package plugin

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin extends a Game: Install wires an ECS module, setup, renderers, and/or resources as one unit.
type Plugin interface {
	// Name uniquely identifies this plugin — Use rejects a duplicate.
	Name() string

	// Install queues this plugin's ECS wiring — dependencies on other
	// plugins are resolved before this, via constructor injection.
	Install(ctx Installer) error

	// RunPlan runs this plugin's per-tick work — call from your own Game.RunPlan, in whatever order you need.
	RunPlan(ctx goke.RunCtx, d time.Duration)

	// WithRenderer configures this plugin's own render.Renderer to draw cam-relative sprites from atlas — call before Use. A no-op for a plugin with no renderer of its own.
	WithRenderer(cam camera.Camera, atlas render.AtlasSource)

	// Renderer returns this plugin's own render.Renderer, or nil if it has none.
	Renderer() render.Renderer

	// EventHandler returns this plugin's own control.EventHandler, or nil if it has none.
	EventHandler() control.EventHandler

	// Serializable returns this plugin's persistable state, or nil if it has none.
	Serializable() Serializable
}
