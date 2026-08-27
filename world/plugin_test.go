package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/world"
	"github.com/kjkrol/gokg"
)

func TestPlugin_Install_PublishesSpaceResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	space, ok := game.Resources().TryGet[*gokg.Space]()
	if !ok {
		t.Fatal("expected *gokg.Space to be registered as a resource")
	}
	if space != plugin.World().Space() {
		t.Error("registered *gokg.Space resource is not plugin.World().Space()")
	}
}

func TestPlugin_Install_DoesNotPublishModuleResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	plugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	if _, ok := game.Resources().TryGet[*world.Module](); ok {
		t.Error("expected *world.Module to NOT be registered as a resource")
	}
}

func TestPlugin_Install_PublishesConfigResource(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	cfg := world.Config{Width: 100, Height: 100, Toroidal: true}
	plugin := world.NewPlugin(cfg, world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10})

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
	plugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	)

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

func TestPlugin_Name(t *testing.T) {
	p := world.NewPlugin(world.Config{}, world.Population{})
	if p.Name() != "gokebiten.world" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.world")
	}
}
