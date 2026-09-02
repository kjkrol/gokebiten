package collisions

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestPlugin_Install_StaysPendingWithoutWorld(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := NewPlugin()

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}
	if plugin.collisions != nil {
		t.Error("expected collisions to stay nil — no world.Plugin was ever installed")
	}
}

func TestPlugin_Install_ResolvesWhenRegisteredBeforeWorld(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	collisionsPlugin := NewPlugin()
	if err := game.UsePlugin(collisionsPlugin); err != nil {
		t.Fatalf("UsePlugin(collisionsPlugin): %v", err)
	}
	if collisionsPlugin.collisions != nil {
		t.Fatal("expected collisions to be nil before world.Plugin installs")
	}

	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	if err := game.UsePlugin(worldPlugin); err != nil {
		t.Fatalf("UsePlugin(worldPlugin): %v", err)
	}

	if collisionsPlugin.collisions == nil {
		t.Fatal("expected collisions to be non-nil once world.Plugin installs, even though collisions was registered first")
	}
}

func TestPlugin_Install_BuildsCollisionsAfterWorld(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	if err := game.UsePlugin(worldPlugin); err != nil {
		t.Fatalf("UsePlugin(worldPlugin): %v", err)
	}

	collisionsPlugin := NewPlugin()
	if err := game.UsePlugin(collisionsPlugin); err != nil {
		t.Fatalf("UsePlugin(collisionsPlugin): %v", err)
	}
	if collisionsPlugin.collisions == nil {
		t.Fatal("expected collisions to be non-nil after Install")
	}
}

func TestPlugin_Name(t *testing.T) {
	p := NewPlugin()
	if p.Name() != "gokebiten.collisions" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.collisions")
	}
}
