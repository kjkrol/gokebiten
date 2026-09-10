package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestPlugin_Install_PublishesResources(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	cfg := world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}
	plugin := world.NewPlugin(cfg)

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	res, ok := game.Resources().TryGet[*world.Resources]()
	if !ok {
		t.Fatal("expected *world.Resources to be registered as a resource")
	}
	if res.Config != cfg {
		t.Errorf("registered Resources.Config = %+v, want %+v", res.Config, cfg)
	}
	if res.Telemetry.Count != 0 {
		t.Errorf("Resources.Telemetry.Count = %d, want 0 (nothing populated)", res.Telemetry.Count)
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
