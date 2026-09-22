package world_test

import (
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
)

func ExamplePlugin_Seed() {
	plugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 800, Height: 600},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 8, MaxSize: 8},
	})
	placement := world.NewGridPlacement(800, 600, 8)

	dot := kind.Define[world.Position](plugin.Kinds(), "dot", kind.Spec{
		kind.Load(func(p world.Position) world.Position { return p }),
		kind.Const(world.Velocity{}),
	})

	for i := range 10 {
		plugin.Seed(dot.Entry(placement.Place(i, 10)))
	}

	if err := plugin.Populate(); err != nil {
		panic(err)
	}
}
