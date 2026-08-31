package gokebiten

import (
	"reflect"
	"time"

	"github.com/kjkrol/goke/v3"
)

// GameCtx is the capability surface Game exposes to Plugin.Install.
type GameCtx struct {
	Resources *Resources

	game  *Game
	wrote bool
}

// Require returns the published value of T, or an error if it isn't published yet — Install may be retried until it is.
func (c *GameCtx) Require[T any]() (T, error) {
	if v, ok := c.Resources.TryGet[T](); ok {
		return v, nil
	}
	var zero T
	return zero, &notReadyError{typ: reflect.TypeFor[T]()}
}

// Provide publishes v so other plugins and the rest of the game can read it.
func (c *GameCtx) Provide[T any](v T) {
	c.wrote = true
	c.Resources.insertResource(v)
}

// UseModule registers m's per-tick systems.
func (c *GameCtx) UseModule(m goke.Module) { c.wrote = true; c.game.useModule(m) }

// Setup registers providers' one-time setup systems.
func (c *GameCtx) Setup(providers ...goke.SetupProvider) {
	c.wrote = true
	c.game.setup(providers...)
}

// RegSys registers a single ad-hoc ECS system.
func (c *GameCtx) RegSys(factory func() goke.System) goke.Runnable {
	c.wrote = true
	return c.game.regSys(factory)
}

// RegComp registers ECS component type C.
func (c *GameCtx) RegComp[C any]() goke.CompID { return c.game.ecs.RegComp[C]() }

// ECS returns the underlying goke ECS.
func (c *GameCtx) ECS() *goke.ECS { return c.game.ecs }

// Step returns the game's fixed tick duration.
func (c *GameCtx) Step() time.Duration { return c.game.step }

// notReadyError is returned by GameCtx.Require when T isn't published yet.
type notReadyError struct{ typ reflect.Type }

func (e *notReadyError) Error() string {
	return "gokebiten: requires " + e.typ.String() + ", not yet published"
}
