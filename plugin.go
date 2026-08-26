package gokebiten

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin extends a Game: Install wires an ECS module, setup, renderers, and/or resources as one unit.
type Plugin interface {
	// Name uniquely identifies this plugin — UsePlugin rejects a duplicate.
	Name() string

	// Install runs once, in UsePlugin call order.
	Install(ctx *PluginContext) error
}

// PluginContext is the capability surface Game exposes to Plugin.Install.
type PluginContext struct {
	Resources *Resources

	game *Game
}

// UseModule registers m's per-tick systems.
func (c *PluginContext) UseModule(m goke.Module) { c.game.useModule(m) }

// Setup registers providers' one-time setup systems.
func (c *PluginContext) Setup(providers ...goke.SetupProvider) { c.game.setup(providers...) }

// RenderSequence appends renderers to the draw pipeline.
func (c *PluginContext) RenderSequence(f ...func() render.Renderer) { c.game.RenderSequence(f...) }

// RegSys registers a single ad-hoc ECS system.
func (c *PluginContext) RegSys(factory func() goke.System) goke.Runnable {
	return c.game.regSys(factory)
}

// ECS returns the underlying goke ECS.
func (c *PluginContext) ECS() *goke.ECS { return c.game.ecs }

// Step returns the game's fixed tick duration.
func (c *PluginContext) Step() time.Duration { return c.game.step }

// UsePlugin installs p, giving it a PluginContext, and rejects a duplicate Name.
func (g *Game) UsePlugin(p Plugin) error { return g.pluginManager.install(p) }
