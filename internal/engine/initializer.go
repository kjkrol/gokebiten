package engine

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// initializer is the only concrete implementation of game.Initializer. It
// binds to one Stage's ecsHost — a fresh initializer is built each time
// Engine enters a Stage.
type initializer struct {
	host  *ecsHost
	world *world.Plugin
	tps   *game.TPS
}

var _ game.Initializer = (*initializer)(nil)

func (c *initializer) UseModule(m goke.Module) { c.host.useModule(m) }

func (c *initializer) Setup(providers ...goke.SetupProvider) { c.host.setup(providers...) }

func (c *initializer) RegSys(factory func() goke.System) goke.Runnable {
	return c.host.regSys(factory)
}

func (c *initializer) ECS() *goke.ECS { return c.host.ecs }

// Use registers p's Serializable (if any) and runs its Install — rejects
// a duplicate Name and any Plugin the engine already installs itself.
func (c *initializer) Use(p plugin.Plugin) error {
	if _, ok := p.(plugin.Builtin); ok {
		return fmt.Errorf("gokebiten: %q is installed automatically by the engine — do not Use it yourself", p.Name())
	}
	return c.use(p)
}

// useBuiltin installs an engine-managed Plugin, bypassing the Builtin
// check above — called only by Engine's own bootstrap, before Stage.Init runs.
func (c *initializer) useBuiltin(p plugin.Plugin) error { return c.use(p) }

// use is the shared install path: duplicate-Name check, Serializable
// registration, tracking, Install.
func (c *initializer) use(p plugin.Plugin) error {
	if c.host.names == nil {
		c.host.names = make(map[string]bool)
	}
	if c.host.names[p.Name()] {
		return fmt.Errorf("gokebiten: plugin %q already used", p.Name())
	}
	c.host.names[p.Name()] = true
	if s := p.Serializable(); s != nil {
		c.host.resources.register(p.Name(), s)
	}
	c.host.track(p)
	return p.Install(c)
}

// Track registers s into the same name-keyed Save/Load machinery as a
// Plugin's own persisted state (by its Go type name — see
// ecsHost.saveTargets), without requiring a full plugin.Plugin.
func (c *initializer) Track(s plugin.Serializable) error {
	c.host.track(s)
	return nil
}

func (c *initializer) World() *world.Plugin { return c.world }

func (c *initializer) TPS() *game.TPS { return c.tps }
