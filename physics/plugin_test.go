package physics_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/physics"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/stats"
	"github.com/kjkrol/gokebiten/world"
)

func TestPlugin_Install_StaysPendingWithoutSpaceResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := physics.NewPlugin(10)

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}
	if plugin.Physics() != nil {
		t.Error("expected Physics() to stay nil — no *gokg.Space resource was ever published")
	}
}

func TestPlugin_Install_ResolvesWhenRegisteredBeforeWorld(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	physicsPlugin := physics.NewPlugin(10)
	if err := game.UsePlugin(physicsPlugin); err != nil {
		t.Fatalf("UsePlugin(physicsPlugin): %v", err)
	}
	if physicsPlugin.Physics() != nil {
		t.Fatal("expected Physics() to be nil before world.Plugin installs")
	}

	worldPlugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)
	if err := game.UsePlugin(worldPlugin); err != nil {
		t.Fatalf("UsePlugin(worldPlugin): %v", err)
	}

	if physicsPlugin.Physics() == nil {
		t.Fatal("expected Physics() to be non-nil once world.Plugin installs, even though physics was registered first")
	}
}

func TestPlugin_Install_BuildsPhysicsAfterSpatialPlugin(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	worldPlugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)
	if err := game.UsePlugin(worldPlugin); err != nil {
		t.Fatalf("UsePlugin(worldPlugin): %v", err)
	}

	physicsPlugin := physics.NewPlugin(10)
	if err := game.UsePlugin(physicsPlugin); err != nil {
		t.Fatalf("UsePlugin(physicsPlugin): %v", err)
	}
	if physicsPlugin.Physics() == nil {
		t.Fatal("expected Physics() to be non-nil after Install")
	}
}

func TestPlugin_EnableStats_PublishesStatsResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	worldPlugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)
	if err := game.UsePlugin(worldPlugin); err != nil {
		t.Fatalf("UsePlugin(worldPlugin): %v", err)
	}

	physicsPlugin := physics.NewPlugin(10).EnableStats()
	if err := game.UsePlugin(physicsPlugin); err != nil {
		t.Fatalf("UsePlugin(physicsPlugin): %v", err)
	}

	got, ok := game.Resources().TryGet[*stats.Stats]()
	if !ok {
		t.Fatal("expected *stats.Stats to be registered as a resource")
	}
	if got != physicsPlugin.Stats() {
		t.Error("registered *stats.Stats resource is not physicsPlugin.Stats()")
	}
}

func TestPlugin_Stats_NilWithoutEnableStats(t *testing.T) {
	p := physics.NewPlugin(10)
	if p.Stats() != nil {
		t.Error("expected Stats() to be nil before EnableStats")
	}
}

func TestPlugin_Name(t *testing.T) {
	p := physics.NewPlugin(10)
	if p.Name() != "gokebiten.physics" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.physics")
	}
}
