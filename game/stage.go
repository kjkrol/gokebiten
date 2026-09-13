package game

import (
	"time"

	"github.com/kjkrol/goke/v3"
)

// Stage is one self-contained context a Game can be in, with its own plugins and ECS.
type Stage interface {
	// Name uniquely identifies this stage.
	Name() string

	// Init runs once, when this Stage becomes active.
	Init(ctx Initializer) error

	// Restore resumes from a save, or reports false if there's nothing to restore.
	Restore(p Persistence) (restored bool, err error)

	// Spawn runs once, only if Restore found nothing, seeding each plugin's initial state (e.g. world.Plugin.Seed).
	Spawn() error

	// Update advances this stage's simulation by d, once per tick.
	Update(ctx goke.RunCtx, d time.Duration)

	// Stack lists every Scene this stage can show, and owns their live Composition.
	Stack() Stack
}
