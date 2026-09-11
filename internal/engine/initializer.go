package engine

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// initializer is the only concrete implementation of game.Initializer.
type initializer struct{ engine *Engine }

var _ game.Initializer = (*initializer)(nil)

func (c *initializer) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(si *goke.SysInit) { m.RegSystems(c.engine.ecs) }}
	c.engine.track(m)
	c.engine.addPendingSetup(func() []goke.System { return append(m.SetupSystems(), regSys) })
}

func (c *initializer) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.engine.track(p)
		c.engine.addPendingSetup(p.SetupSystems)
	}
}

func (c *initializer) RegSys(factory func() goke.System) goke.Runnable {
	return c.engine.ecs.RegSys(factory())
}

func (c *initializer) ECS() *goke.ECS { return c.engine.ecs }

// Use registers p's Serializable (if any) and runs its Install — rejects
// a duplicate Name and any Plugin the engine already installs itself.
func (c *initializer) Use(p plugin.Plugin) error {
	if _, ok := p.(plugin.Builtin); ok {
		return fmt.Errorf("gokebiten: %q is installed automatically by the engine — do not Use it yourself", p.Name())
	}
	return c.use(p)
}

// useBuiltin installs an engine-managed Plugin, bypassing the Builtin
// check above — called only by Engine's own bootstrap, before Game.Init runs.
func (c *initializer) useBuiltin(p plugin.Plugin) error { return c.use(p) }

// use is the shared install path: duplicate-Name check, Serializable
// registration, tracking, Install.
func (c *initializer) use(p plugin.Plugin) error {
	if c.engine.names == nil {
		c.engine.names = make(map[string]bool)
	}
	if c.engine.names[p.Name()] {
		return fmt.Errorf("gokebiten: plugin %q already used", p.Name())
	}
	c.engine.names[p.Name()] = true
	if s := p.Serializable(); s != nil {
		c.engine.resources.register(p.Name(), s)
	}
	c.engine.track(p)
	return p.Install(c)
}

func (c *initializer) World() *world.Plugin { return c.engine.world }
