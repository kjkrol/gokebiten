package world_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
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

	space, ok := game.Resources().TryGetResource[*gokg.Space]()
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

	if _, ok := game.Resources().TryGetResource[*world.Module](); ok {
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

	got, ok := game.Resources().TryGetResource[world.Config]()
	if !ok {
		t.Fatal("expected world.Config to be registered as a resource")
	}
	if got != cfg {
		t.Errorf("registered world.Config = %+v, want %+v", got, cfg)
	}
}

func TestPlugin_OnReindexed_ThreadsToModule(t *testing.T) {
	game := gokebiten.NewGame(&gokebiten.GameProps{})
	var gotCount int
	called := false
	plugin := world.NewPlugin(
		world.Config{Width: 100, Height: 100},
		world.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	).OnReindexed(func(count int) { called = true; gotCount = count })

	if err := game.UsePlugin(plugin); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	goke.New().Setup(plugin.World().PostLoad())

	if !called {
		t.Fatal("expected OnReindexed's callback to be called via Module.PostLoad")
	}
	if gotCount != 0 {
		t.Errorf("callback count = %d, want 0 (no entities loaded)", gotCount)
	}
}

func TestPlugin_Name(t *testing.T) {
	p := world.NewPlugin(world.Config{}, world.Population{})
	if p.Name() != "gokebiten.world" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.world")
	}
}
