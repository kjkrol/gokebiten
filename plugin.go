package gokebiten

import (
	"reflect"
	"time"

	"github.com/kjkrol/goke/v3"
)

// Plugin extends a Game: Install wires an ECS module, setup, renderers, and/or resources as one unit.
type Plugin interface {
	// Name uniquely identifies this plugin — UsePlugin rejects a duplicate.
	Name() string

	// Install runs once its dependencies are available — return an error from Require to be retried automatically.
	Install(ctx *GameCtx) error
}

// GameCtx is the capability surface Game exposes to Plugin.Install.
type GameCtx struct {
	Resources *Resources

	game  *Game
	wrote bool
}

// Require returns the published value of T, or an error if it isn't published yet — Install may be retried until it is.
func (c *GameCtx) Require[T any]() (T, error) {
	if v, ok := c.Resources.TryGetResource[T](); ok {
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

// ECS returns the underlying goke ECS.
func (c *GameCtx) ECS() *goke.ECS { return c.game.ecs }

// Step returns the game's fixed tick duration.
func (c *GameCtx) Step() time.Duration { return c.game.step }

// UsePlugin installs p once its dependencies are available, retrying automatically as other plugins install, and rejects a duplicate Name.
func (g *Game) UsePlugin(p Plugin) error { return g.pluginManager.install(p) }

// Init runs fn to build the game's own logic — only one is allowed per Game.
func (g *Game) Init(fn func(ctx *GameCtx) error) error {
	return g.UsePlugin(&gameInitPlugin{fn: fn})
}

// notReadyError is returned by GameCtx.Require when T isn't published yet.
type notReadyError struct{ typ reflect.Type }

func (e *notReadyError) Error() string {
	return "gokebiten: requires " + e.typ.String() + ", not yet published"
}

// gameInitPlugin adapts a plain closure to Plugin — see Game.Init.
type gameInitPlugin struct{ fn func(ctx *GameCtx) error }

func (p *gameInitPlugin) Name() string { return "gokebiten.game" }

func (p *gameInitPlugin) Install(ctx *GameCtx) error { return p.fn(ctx) }
