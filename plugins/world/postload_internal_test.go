package world

import (
	"testing"

	"github.com/kjkrol/goke/v3"
)

func testModule() *Module {
	return NewModule(Config{
		Space:    SpaceCfg{Width: 1000, Height: 1000},
		Entities: EntitiesCfg{MaxCount: 10, MinSize: 1, MaxSize: 100},
	})
}

func TestModule_PostLoad_SetsCountToZeroWhenNothingSeeded(t *testing.T) {
	w := testModule()

	goke.New().Setup(w.PostLoad())

	if w.telemetry.Count != 0 {
		t.Errorf("telemetry.Count = %d, want 0 (no Position entities seeded)", w.telemetry.Count)
	}
}

func TestModule_PostLoad_SetsCountFromLoadedEntities(t *testing.T) {
	w := testModule()

	pos := Position{}
	pos.Size.X, pos.Size.Y = 10, 10

	var posComp goke.Comp[Position]
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

	if w.telemetry.Count != 3 {
		t.Errorf("telemetry.Count = %d, want 3", w.telemetry.Count)
	}
}
