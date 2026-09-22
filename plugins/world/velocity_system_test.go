package world_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/world"
)

// constFactorModifier is a world.SpeedModifier test double that scales
// every entity's Velocity by a fixed factor, binding no extra components.
type constFactorModifier struct{ factor float64 }

func (m constFactorModifier) Bind(*goke.QueryBuilder) {}
func (m constFactorModifier) Apply(_ *goke.Cursor, _ int, _ *world.Base, acc float64) float64 {
	return acc * m.factor
}

func TestVelocitySystem_Update_ComposesModifiersMultiplicatively(t *testing.T) {
	ecs := goke.New()
	var baseComp goke.Comp[world.Base]
	var q *goke.Query
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&baseComp)
		f.Create(1)
		f.Next()
		baseComp.Slice(&f.Cursor)[0].Vel = world.Velocity{Dir: geom.NewVec(1, 0), Value: 100}
		q = si.NewQueryBuilder(&baseComp).Build()
	}})

	sys := world.NewVelocitySystem([]world.SpeedModifier{
		constFactorModifier{factor: 0.5},
		constFactorModifier{factor: 0.25},
	})
	handle := ecs.RegSys(sys)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second)

	q.All()
	for q.Next() {
		bases := baseComp.Slice(q.Cursor())
		if len(bases) == 0 {
			continue
		}
		if math.Abs(bases[0].Vel.Value-12.5) > 1e-9 {
			t.Errorf("Velocity.Value = %v, want 12.5 — modifiers should compose multiplicatively", bases[0].Vel.Value)
		}
		return
	}
	t.Fatal("expected to find the seeded entity")
}
