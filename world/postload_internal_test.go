package world

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
)

func testModule() *Module {
	return NewModule(
		Config{Width: 1000, Height: 1000},
		Population{MaxCount: 10, MinSize: 1, MaxSize: 100},
	)
}

func TestModule_PostLoad_SetsDynamicCountToZeroWhenNothingSeeded(t *testing.T) {
	w := testModule()

	goke.New().Setup(w.PostLoad())

	if w.telemetry.DynamicCount != 0 {
		t.Errorf("telemetry.DynamicCount = %d, want 0 (no kinematics.Position entities seeded)", w.telemetry.DynamicCount)
	}
}

func TestModule_PostLoad_SetsDynamicCountFromLoadedEntities(t *testing.T) {
	w := testModule()

	pos := kinematics.Position{}
	pos.Size.X, pos.Size.Y = 10, 10

	var posComp goke.Comp[kinematics.Position]
	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&posComp)
		f.Create(3)
		for f.Next() {
			for i := range posComp.Slice(&f.Cursor) {
				posComp.Slice(&f.Cursor)[i] = pos
			}
		}
	}}, w.PostLoad())

	if w.telemetry.DynamicCount != 3 {
		t.Errorf("telemetry.DynamicCount = %d, want 3", w.telemetry.DynamicCount)
	}
}
