package kinematics_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokebiten/spatial"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// newTestWorld seeds one entity with the given starting Velocity and
// returns the ECS (with maxSpeed's kinematics.System registered and
// plan set), a verification query, and the seeded entity's Position
// component handle for reading AccX/TopLeft after ticking.
func newTestWorld(t *testing.T, maxSpeed int32, vel kinematics.Velocity) (*goke.ECS, *goke.Query, *goke.Comp[kinematics.Position]) {
	t.Helper()
	space := spatial.NewWorldModule(
		spatial.Config{Width: 1000, Height: 1000},
		spatial.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	).Space()

	ecs := goke.New()
	var pos goke.Comp[kinematics.Position]
	var velComp goke.Comp[kinematics.Velocity]
	var q *goke.Query
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&pos, &velComp)
		f.Create(1)
		f.Next()
		positions := pos.Slice(&f.Cursor)
		velocities := velComp.Slice(&f.Cursor)
		positions[0] = kinematics.Position{AABB: plane.NewAABB(geom.NewVec[uint32](0, 0), 5, 5)}
		velocities[0] = vel

		q = si.NewQueryBuilder(&pos).Build()
	}})

	sys := kinematics.NewSystem(space, maxSpeed)
	handle := ecs.RegSys(sys)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})

	return ecs, q, &pos
}

func readFirst(t *testing.T, q *goke.Query, pos *goke.Comp[kinematics.Position]) kinematics.Position {
	t.Helper()
	q.All()
	for q.Next() {
		s := pos.Slice(q.Cursor())
		if len(s) > 0 {
			return s[0]
		}
	}
	t.Fatal("expected to find the seeded entity")
	return kinematics.Position{}
}

func TestSystem_Update_SubPixelAccumulatesWithoutMoving(t *testing.T) {
	vel := kinematics.Velocity{}
	vel.Vec.X = 1 // 1 unit/sec
	ecs, q, pos := newTestWorld(t, 0, vel)

	ecs.Tick(10 * time.Millisecond) // AccX += 1*0.01 = 0.01, dx=0

	p := readFirst(t, q, pos)
	if p.TopLeft.X != 0 {
		t.Errorf("TopLeft.X = %d, want 0 (sub-pixel movement shouldn't translate)", p.TopLeft.X)
	}
	if p.AccX < 0.009 || p.AccX > 0.011 {
		t.Errorf("AccX = %v, want ~0.01 (the sub-pixel remainder should accumulate)", p.AccX)
	}
}

func TestSystem_Update_TranslatesOnWholePixels(t *testing.T) {
	vel := kinematics.Velocity{}
	vel.Vec.X = 100 // 100 units/sec
	ecs, q, pos := newTestWorld(t, 0, vel)

	for range 3 {
		ecs.Tick(20 * time.Millisecond) // AccX += 100*0.02 = 2.0 exactly, each tick
	}

	p := readFirst(t, q, pos)
	if p.TopLeft.X != 6 {
		t.Errorf("TopLeft.X = %d, want 6 (3 ticks x 2px, no remainder)", p.TopLeft.X)
	}
	if p.AccX != 0 {
		t.Errorf("AccX = %v, want 0 (each tick's 2.0 was a whole number, nothing left over)", p.AccX)
	}
}

func TestSystem_Update_ClampsSpeedBeforeIntegrating(t *testing.T) {
	vel := kinematics.Velocity{}
	vel.Vec.X = 1000 // far above maxSpeed
	ecs, q, pos := newTestWorld(t, 10, vel)

	ecs.Tick(time.Second) // clamped to 10, AccX += 10*1 = 10

	p := readFirst(t, q, pos)
	if p.TopLeft.X != 10 {
		t.Errorf("TopLeft.X = %d, want 10 (velocity should have been clamped to maxSpeed before integrating)", p.TopLeft.X)
	}
}
