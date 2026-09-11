package game

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Game is implemented by the user — Engine drives it.
type Game interface {
	// Init runs once: construct/configure plugins via ctx.Use.
	Init(ctx Initializer) error

	// Restore attempts to resume from a save. restored is false (with a
	// nil error) when there's nothing to restore — Engine then calls Spawn.
	Restore(p Persistence) (restored bool, err error)

	// Spawn runs once, only when Restore returns false: returns the
	// initial entity batches for Engine to Populate.
	Spawn() ([]world.Batch, error)

	// Update advances this game's simulation by d — the engine calls it once per tick.
	Update(ctx goke.RunCtx, d time.Duration)

	// Draw's renderers run bottom-to-top every cycle to refresh the screen.
	Draw(runtime Runtime) []func() render.Renderer

	// HandleEvents runs this tick's input handling — runtime is fresh each call.
	HandleEvents(events *control.InputEvents, runtime Runtime)
}
