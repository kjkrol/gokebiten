package engine

import (
	"fmt"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
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

// Use registers p's Serializable (if any) and runs its Install — rejects a duplicate Name.
func (c *initializer) Use(p plugin.Plugin) error {
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

func (c *initializer) Runtime() game.Runtime { return c.engine }
