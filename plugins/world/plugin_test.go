package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestPlugin_Install_DoesNotPublishWorldResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	if _, ok := game.Resources().TryGet[*world.World](); ok {
		t.Error("expected *world.World to NOT be registered as a resource")
	}
}

func TestPlugin_Install_PublishesConfigResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	cfg := world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}
	plugin := world.NewPlugin(cfg)

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	got, ok := game.Resources().TryGet[world.Config]()
	if !ok {
		t.Fatal("expected world.Config to be registered as a resource")
	}
	if got != cfg {
		t.Errorf("registered world.Config = %+v, want %+v", got, cfg)
	}
}

func TestPlugin_Install_PublishesTelemetryResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	telemetry, ok := game.Resources().TryGet[*world.Telemetry]()
	if !ok {
		t.Fatal("expected *world.Telemetry to be registered as a resource")
	}
	if telemetry != plugin.World().Telemetry() {
		t.Error("registered *world.Telemetry resource is not plugin.World().Telemetry()")
	}
}

func TestPlugin_Install_PublishesItself(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	got, ok := game.Resources().TryGet[*world.Plugin]()
	if !ok {
		t.Fatal("expected *world.Plugin to be registered as a resource — other plugins need it to call RegisterSpeedModifier")
	}
	if got != plugin {
		t.Error("registered *world.Plugin resource is not the installed plugin")
	}
}

func TestPlugin_Name(t *testing.T) {
	p := world.NewPlugin(world.Config{})
	if p.Name() != "gokebiten.world" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.world")
	}
}
