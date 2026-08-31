package world_test

import (
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
		velComp.Slice(&f.Cursor)[0] = world.Velocity{Dir: geom.NewVec[float64](1, 0), Value: 100}
		q = si.NewQueryBuilder(&velComp).Build()
	}})

	sys := world.NewVelocitySystem([]world.SpeedModifier{
		constFactorModifier{factor: 0.5},
		constFactorModifier{factor: 0.25},
	}, 0)
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
		// 100 * 0.5 * 0.25 = 12.5, truncated to 12 — the two factors must
		// have been multiplied together, not just the last one applied.
		if vel[0].Value != 12 {
			t.Errorf("Velocity.Value = %d, want 12 — modifiers should compose multiplicatively", vel[0].Value)
		}
		return
	}
	t.Fatal("expected to find the seeded entity")
}

func TestVelocitySystem_Update_ClampsToMaxSpeed(t *testing.T) {
	ecs := goke.New()
	var velComp goke.Comp[world.Velocity]
	var q *goke.Query
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&velComp)
		f.Create(1)
		f.Next()
		velComp.Slice(&f.Cursor)[0] = world.Velocity{Dir: geom.NewVec[float64](1, 0), Value: 1000}
		q = si.NewQueryBuilder(&velComp).Build()
	}})

	sys := world.NewVelocitySystem(nil, 10)
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
		if vel[0].Value != 10 {
			t.Errorf("Velocity.Value = %d, want 10 (clamped to maxSpeed)", vel[0].Value)
		}
		return
	}
	t.Fatal("expected to find the seeded entity")
}
