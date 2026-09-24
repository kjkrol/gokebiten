package world_test

import (
	"strings"
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/gram/plugins/world/kind/comp"
)

type flatGround struct{ level float64 }

func (g flatGround) At(geom.Vec) float64 { return g.level }
func (g flatGround) Step() float64       { return 32 }

func TestQuasi3D_IsTheConfigsChoiceAndTheGroundIsWhatTheBoardSets(t *testing.T) {
	cfg := world.Config{Space: world.SpaceCfg{Width: 100, Height: 100}, Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 10, MaxSize: 10}}
	flat := world.NewPlugin(cfg)
	if flat.Quasi3D() || flat.Ground() != nil {
		t.Errorf("a world by default: Quasi3D %v, Ground %v; want flat and no ground", flat.Quasi3D(), flat.Ground())
	}
	cfg.Quasi3D = true
	tall := world.NewPlugin(cfg)
	tall.SetGround(flatGround{level: 5})
	if !tall.Quasi3D() || tall.Ground() == nil || tall.Ground().At(geom.NewVec(1, 1)) != 5 {
		t.Errorf("a Quasi3D world with ground set: Quasi3D %v, Ground %v", tall.Quasi3D(), tall.Ground())
	}
}

func TestKinds_RefuseAZInAFlatWorld(t *testing.T) {
	w := world.NewPlugin(world.Config{Space: world.SpaceCfg{Width: 100, Height: 100}, Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 10, MaxSize: 10}})
	defer func() {
		if msg, _ := recover().(string); !strings.Contains(msg, "Quasi3D") {
			t.Errorf("panic %q, want one pointing at world.Config.Quasi3D", msg)
		}
	}()
	kind.Define[struct{}](w.Kinds(), "tower", kind.Spec{
		comp.Const(world.Position{}), comp.Const(world.Velocity{}), comp.Const(world.Z{Height: 3}),
	})
	t.Error("a flat world took a kind carrying a Z")
}
