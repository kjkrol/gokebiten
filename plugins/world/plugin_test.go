package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestPlugin_Res_PublishesConfig(t *testing.T) {
	cfg := world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}
	plugin := world.NewPlugin(cfg)

	if plugin.Res.Config != cfg {
		t.Errorf("Res.Config = %+v, want %+v", plugin.Res.Config, cfg)
	}
	if plugin.Res.Telemetry.Count != 0 {
		t.Errorf("Res.Telemetry.Count = %d, want 0 (nothing populated)", plugin.Res.Telemetry.Count)
	}
}
