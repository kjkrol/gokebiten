package world_test

import (
	"github.com/kjkrol/gokebiten/plugins/world"
)

// ExamplePlugin_Seed shows the shape of a spawn: Define declares a kind whose
// components are fixed (Const) or read from its roster data (k.Load), Entry
// builds roster entries through the dictionary, Seed and Populate spawn them.
func ExamplePlugin_Seed() {
	plugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 800, Height: 600},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 8, MaxSize: 8},
	})
	placement := world.NewGridPlacement(800, 600, 8)

	kinds := plugin.EntKindDict()
	kinds.Define("dot", func(k world.Kind[world.Position]) world.EntKind {
		return world.EntKind{
			Position: k.Load(func(p world.Position) world.Position { return p }),
			Velocity: world.Const(world.Velocity{}),
		}
	})

	for i := range 10 {
		plugin.Seed(kinds.Entry("dot", placement.Place(i, 10)))
	}

	if err := plugin.Populate(); err != nil {
		panic(err)
	}
}
