package world_test

import (
	"github.com/kjkrol/gokebiten/plugins/world"
)

// ExampleModule_Populate shows the shape every Populate call takes: a
// Spawner deciding Position/Velocity, plus one world.NewValueExtras per
// further component — the only implementation of EntityExtras this library
// ships. Attach WithEffect for a side effect (like updating an occupancy
// tracker) that must run alongside the write.
func ExampleModule_Populate() {
	wm := world.NewModule(world.Config{
		Space:    world.SpaceCfg{Width: 800, Height: 600},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 8, MaxSize: 8},
	})
	placement := world.NewGridPlacement(800, 600, 8)

	wm.Populate(10,
		world.SpawnerFunc(func(index, count int) (world.Position, world.Velocity) {
			return placement.Place(index, count), world.Velocity{}
		}),
		world.NewValueExtras(func(index int) world.Appearance {
			return world.Appearance{SpriteID: 0}
		}),
	)
}
