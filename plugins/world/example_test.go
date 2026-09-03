package world_test

import (
	"github.com/kjkrol/gokebiten/plugins/world"
)

// ExampleWorld_Populate shows the shape every Populate call takes:
// NewSpawner requires Position and Velocity, and With attaches each
// further component. Attach WithEffect instead for a side effect (like
// updating an occupancy tracker) that must run alongside the write.
func ExampleWorld_Populate() {
	wm := world.NewWorld(world.Config{
		Space:    world.SpaceCfg{Width: 800, Height: 600},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 8, MaxSize: 8},
	})
	placement := world.NewGridPlacement(800, 600, 8)

	spawner := world.NewSpawner(
		func(index, count int) world.Position { return placement.Place(index, count) },
		func(index int) world.Velocity { return world.Velocity{} },
	).With(func(index int) world.Appearance {
		return world.Appearance{SpriteID: 0}
	})

	wm.Populate(10, spawner)
}
