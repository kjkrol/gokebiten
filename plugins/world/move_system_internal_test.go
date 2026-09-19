package world

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

func TestClampStep(t *testing.T) {
	// clampStep assumes max > 0 — MoveSystem.Update only calls it under that guard.
	cases := []struct {
		name         string
		step         geom.Vec
		max          float64
		wantX, wantY float64
	}{
		{"within max is left alone", geom.NewVec(3, 4), 10, 3, 4},
		{"exactly at max is left alone", geom.NewVec(3, 4), 5, 3, 4},
		{"above max scales down, keeping the direction", geom.NewVec(6, 8), 5, 3, 4},
		{"a standstill has no direction to keep", geom.NewVec(0, 0), 5, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := clampStep(c.step, c.max)
			if math.Abs(got.X-c.wantX) > 1e-12 || math.Abs(got.Y-c.wantY) > 1e-12 {
				t.Errorf("clampStep(%v, %v) = %v, want (%v, %v)", c.step, c.max, got, c.wantX, c.wantY)
			}
		})
	}
}

// This is what the continuous world buys. An entity crossing half a unit per
// tick used to sit still every other tick, because an integer position could
// not hold the half and Velocity carried it in an accumulator instead. Now the
// entity simply moves half a unit, every tick.
func TestMoveSystem_CarriesSubUnitSpeedEveryTick(t *testing.T) {
	const tps = 60
	wm := testWorld()

	// 30 units a second at 60 ticks a second is half a unit per tick.
	wm.populate(EntKind{
		Name:     "e",
		Position: Const(Position{AABB: plane.NewAABB(geom.NewVec(100, 100), 10, 10)}),
		Velocity: Const(Velocity{Dir: geom.NewVec(1, 0), Value: 30}),
	}, []any{nil})

	var pos goke.Comp[Position]
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&pos).Build()
	}})...)
	wm.RegSystems(ecs)
	ecs.SetPlan(wm.RunPlan)

	x := func() float64 {
		query.All()
		for query.Next() {
			return pos.Slice(query.Cursor())[0].TopLeft.X
		}
		t.Fatal("the entity vanished")
		return 0
	}

	// A tick is a whole number of nanoseconds, so 1/60 s is 16666666ns and half
	// a unit is really 0.49999998 — the slack below is for that truncation, not
	// for anything the movement does.
	const step = 30 * (16666666.0 / 1e9)

	start := x()
	for tick := 1; tick <= 4; tick++ {
		ecs.Tick(time.Second / tps)
		want := start + step*float64(tick)
		if got := x(); math.Abs(got-want) > 1e-6 {
			t.Fatalf("after %d ticks the entity is at %v, want %v — half a unit per tick, every tick", tick, got, want)
		}
	}
}
