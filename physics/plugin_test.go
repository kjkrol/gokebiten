package physics_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/physics"
	"github.com/kjkrol/gokebiten/world"
)

func TestPlugin_Install_RequiresSpaceResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := physics.NewPlugin(10)

	err := game.UsePlugin(plugin)
	if err == nil {
		t.Fatal("expected an error when no *gokg.Space resource is registered")
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

func TestPlugin_Name(t *testing.T) {
	p := physics.NewPlugin(10)
	if p.Name() != "gokebiten.physics" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.physics")
	}
}
