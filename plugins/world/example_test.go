package world_test

import (
	"github.com/kjkrol/gokebiten/plugins/world"
)

// ExamplePlugin_Seed shows the shape of a spawn: an EntKind lists every
// component, each fixed (Const) or read from a Roster Entry's Data (Load);
// Seed declares the entities and Populate spawns them.
func ExamplePlugin_Seed() {
	plugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 800, Height: 600},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 8, MaxSize: 8},
	})
	placement := world.NewGridPlacement(800, 600, 8)

	plugin.EntKindDict().Create(world.EntKind{
		Name:     "dot",
		Position: world.Load(func(p world.Position) world.Position { return p }),
		Velocity: world.Const(world.Velocity{}),
	})

	var roster world.Roster
	for i := range 10 {
		roster = append(roster, world.Entry{Kind: "dot", Data: placement.Place(i, 10)})
	}
	plugin.Seed(roster)

	if err := plugin.Populate(); err != nil {
		panic(err)
	}
}
