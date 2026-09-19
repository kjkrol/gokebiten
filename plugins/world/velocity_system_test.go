package world_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// constFactorModifier is a world.SpeedModifier test double that scales
// every entity's Velocity by a fixed factor, binding no extra components.
type constFactorModifier struct{ factor float64 }

func (m constFactorModifier) Bind(*goke.QueryBuilder) {}
func (m constFactorModifier) Apply(_ *goke.Cursor, _ int, acc float64) float64 {
	return acc * m.factor
}

func TestVelocitySystem_Update_ComposesModifiersMultiplicatively(t *testing.T) {
	ecs := goke.New()
	var velComp goke.Comp[world.Velocity]
	var q *goke.Query
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&velComp)
		f.Create(1)
		f.Next()
		velComp.Slice(&f.Cursor)[0] = world.Velocity{Dir: geom.NewVec(1, 0), Value: 100}
		q = si.NewQueryBuilder(&velComp).Build()
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
		vel := velComp.Slice(q.Cursor())
		if len(vel) == 0 {
			continue
		}
		// 100 * 0.5 * 0.25 = 12.5, and it stays 12.5 — a continuous speed no
		// longer loses the half to truncation. The two factors must have been
		// multiplied together, not just the last one applied.
		if math.Abs(vel[0].Value-12.5) > 1e-9 {
			t.Errorf("Velocity.Value = %v, want 12.5 — modifiers should compose multiplicatively", vel[0].Value)
		}
		return
	}
	t.Fatal("expected to find the seeded entity")
}
