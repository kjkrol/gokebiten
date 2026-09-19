package world_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/gokg/spatial"
)

func testSpace(t *testing.T) *gokg.Space {
	t.Helper()
	space, err := gokg.NewSpace(gokg.Config{
		Width: 1000, Height: 1000,
		BucketSize: spatial.ResolutionFrom(64), BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("gokg.NewSpace: %v", err)
	}
	return space
}

// testHandles bundles the component handles a test needs to read back after ticking.
type testHandles struct {
	pos *goke.Comp[world.Position]
	vel *goke.Comp[world.Velocity]
}

// newTestWorld seeds one entity with the given starting Velocity and
// returns the ECS (with world.MoveSystem registered and plan set), a
// verification query, and the seeded entity's component handles.
// maxDelta caps per-tick displacement (0 for no limit).
func newTestWorld(t *testing.T, vel world.Velocity, maxDelta float64) (*goke.ECS, *goke.Query, testHandles) {
	t.Helper()
	space := testSpace(t)

	ecs := goke.New()
	var pos goke.Comp[world.Position]
	var velComp goke.Comp[world.Velocity]
	var q *goke.Query
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&pos, &velComp)
		f.Create(1)
		f.Next()
		positions := pos.Slice(&f.Cursor)
		velocities := velComp.Slice(&f.Cursor)
		positions[0] = world.Position{AABB: plane.NewAABB(geom.NewVec(0, 0), 5, 5)}
		velocities[0] = vel

		q = si.NewQueryBuilder(&pos, &velComp).Build()
	}})

	sys := world.NewMoveSystem(space, maxDelta)
	handle := ecs.RegSys(sys)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})

	return ecs, q, testHandles{pos: &pos, vel: &velComp}
}

func readFirst(t *testing.T, q *goke.Query, h testHandles) (world.Position, world.Velocity) {
	t.Helper()
	q.All()
	for q.Next() {
		p := h.pos.Slice(q.Cursor())
		v := h.vel.Slice(q.Cursor())
		if len(p) > 0 {
			return p[0], v[0]
		}
	}
	t.Fatal("expected to find the seeded entity")
	return world.Position{}, world.Velocity{}
}

func TestMoveSystem_Update_SubPixelAccumulatesWithoutMoving(t *testing.T) {
	vel := world.Velocity{Dir: geom.NewVec(1, 0), Value: 1} // 1 unit/sec
	ecs, q, h := newTestWorld(t, vel, 0)

	ecs.Tick(10 * time.Millisecond) // 1 * 0.01 = one hundredth of a unit

	p, _ := readFirst(t, q, h)
	if math.Abs(p.TopLeft.X-0.01) > 1e-9 {
		t.Errorf("TopLeft.X = %v, want 0.01 — the step is a hundredth of a unit and it should land there", p.TopLeft.X)
	}
}

func TestMoveSystem_Update_TranslatesWholeUnits(t *testing.T) {
	vel := world.Velocity{Dir: geom.NewVec(1, 0), Value: 100} // 100 units/sec
	ecs, q, h := newTestWorld(t, vel, 0)

	for range 3 {
		ecs.Tick(20 * time.Millisecond) // AccX += 100*0.02 = 2.0 exactly, each tick
	}

	p, _ := readFirst(t, q, h)
	if math.Abs(p.TopLeft.X-6) > 1e-9 {
		t.Errorf("TopLeft.X = %v, want 6 (three ticks of two units)", p.TopLeft.X)
	}
}

func TestMoveSystem_Update_ClampsDisplacementToMaxDelta(t *testing.T) {
	vel := world.Velocity{Dir: geom.NewVec(1, 0), Value: 1000} // 1000 units/sec
	ecs, q, h := newTestWorld(t, vel, 3)                       // cap: 3 units/tick

	ecs.Tick(100 * time.Millisecond) // uncapped the step would be 100 units

	p, _ := readFirst(t, q, h)
	if math.Abs(p.TopLeft.X-3) > 1e-9 {
		t.Errorf("TopLeft.X = %v, want 3 (the step is clamped to maxDelta)", p.TopLeft.X)
	}
}
