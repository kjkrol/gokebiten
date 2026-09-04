package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

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
	if telemetry.Count != 0 {
		t.Errorf("Telemetry.Count = %d, want 0 (nothing populated)", telemetry.Count)
	}
}

func TestPlugin_Name(t *testing.T) {
	p := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	if p.Name() != "gokebiten.world" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.world")
	}
}
