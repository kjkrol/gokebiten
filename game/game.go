package game

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/render"
)

// Game is implemented by the user — Engine drives it.
type Game interface {
	// Init runs once: construct/configure plugins via ctx.Use, then spawn fresh entities or restore a save.
	Init(ctx Initializer) error

	// RunPlan runs this game's per-tick simulation order.
	RunPlan(ctx goke.RunCtx, d time.Duration)

	// Layers returns the draw layers, bottom to top — called once, after Init.
	Layers(runtime Runtime) []func() render.Renderer

	// HandleEvents runs this tick's input handling — runtime is fresh each call.
	HandleEvents(events *control.InputEvents, runtime Runtime)
}
