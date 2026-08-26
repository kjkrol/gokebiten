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

func TestModule_PostLoad_CallsOnReindexedWithZeroCount(t *testing.T) {
	w := testModule()

	var gotCount int
	called := false
	w.onReindexed = func(count int) { called = true; gotCount = count }

	goke.New().Setup(w.PostLoad())

	if !called {
		t.Fatal("expected onReindexed to be called")
	}
	if gotCount != 0 {
		t.Errorf("onReindexed count = %d, want 0 (no kinematics.Position entities seeded)", gotCount)
	}
}

func TestModule_PostLoad_ReindexesLoadedEntities(t *testing.T) {
	w := testModule()

	var gotCount int
	w.onReindexed = func(count int) { gotCount = count }

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

	if gotCount != 3 {
		t.Errorf("onReindexed count = %d, want 3", gotCount)
	}
}
