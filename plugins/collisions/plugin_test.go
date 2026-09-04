package collisions

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
)

func testWorldPlugin() *world.Plugin {
	return world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
}

func TestPlugin_Name(t *testing.T) {
	p := NewPlugin(0, testWorldPlugin())
	if p.Name() != "gokebiten.collisions" {
		t.Errorf("Name() = %q, want %q", p.Name(), "gokebiten.collisions")
	}
}
