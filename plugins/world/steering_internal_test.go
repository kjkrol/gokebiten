package world

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world/kind"
)

var (
	east  = geom.NewVec(1.0, 0.0)
	north = geom.NewVec(0.0, 1.0)
)

// steerTicks spawns one entity heading start, carrying st, and reports its heading per tick.
func steerTicks(t *testing.T, st Steering, start geom.Vec, n int) []geom.Vec {
	t.Helper()

	wm := testWorld()
	wm.populate(testKind(
		Position{AABB: plane.NewAABB(geom.NewVec(500, 500), 10, 10)},
		Velocity{Dir: start, Value: 1},
		kind.Const(st),
	), []any{nil})

	var base goke.Comp[Base]
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&base).Build()
	}})...)
	wm.RegSystems(ecs)
	ecs.SetPlan(wm.RunPlan)

	var out []geom.Vec
	for range n {
		ecs.Tick(time.Second / 60)
		query.All()
		for query.Next() {
			for _, b := range base.Slice(query.Cursor()) {
				out = append(out, b.Vel.Dir)
			}
		}
	}
	return out
}

func heading(v geom.Vec) float64 { return math.Atan2(v.Y, v.X) }

func TestSteering_RequestIsRefusedWhileStillReacting(t *testing.T) {
	s := &Steering{Reflex: 3}

	if !s.Request(east) {
		t.Fatal("first Request refused on an idle Steering")
	}
	if s.Request(north) {
		t.Error("second Request taken while the first was still being reacted to")
	}
	if s.Pending != east {
		t.Errorf("Pending = %v, want the first request kept", s.Pending)
	}
}

func TestSteering_RequestNormalisesWhateverItIsHanded(t *testing.T) {
	s := &Steering{}
	s.Request(geom.NewVec(3.0, 4.0))

	if n := math.Hypot(s.Want.X, s.Want.Y); math.Abs(n-1) > 1e-12 {
		t.Errorf("|Want| = %v, want 1", n)
	}
	if math.Abs(heading(s.Want)-math.Atan2(4, 3)) > 1e-12 {
		t.Errorf("Want points at %v, want the direction it was handed", heading(s.Want))
	}
}

func TestSteering_ReflexHoldsTheTurnBack(t *testing.T) {
	dirs := steerTicks(t, Steering{Pending: east, Reflex: 2, Delay: 2}, north, 3)

	if dirs[0] != north || dirs[1] != north {
		t.Errorf("headings %v, %v during the reflex window, want both still north", dirs[0], dirs[1])
	}
	if dirs[2] != east {
		t.Errorf("heading %v once the reflex ran out, want east", dirs[2])
	}
}

func TestSteering_TurnRateCapsTheSwing(t *testing.T) {
	const rate = 0.1
	dirs := steerTicks(t, Steering{Want: east, TurnRate: rate}, north, 3)

	from := heading(north)
	for i, d := range dirs {
		want := from - rate*float64(i+1)
		if math.Abs(heading(d)-want) > 1e-9 {
			t.Errorf("tick %d heading %.4f, want %.4f — one rate step per tick", i+1, heading(d), want)
		}
	}
	if heading(dirs[2]) <= 0 {
		t.Error("reached the target within three ticks, so the rate was not capping anything")
	}
}

func TestSteering_NoRateSwingsAllTheWayAtOnce(t *testing.T) {
	if dirs := steerTicks(t, Steering{Want: east}, north, 1); dirs[0] != east {
		t.Errorf("heading %v after one tick with no TurnRate, want east", dirs[0])
	}
}

// The last step lands on the target exactly rather than overshooting it.
func TestSteering_LastStepSettlesOnTheTarget(t *testing.T) {
	dirs := steerTicks(t, Steering{Want: east, TurnRate: 1.0}, north, 2)

	if heading(dirs[0]) <= 0 {
		t.Fatalf("heading %.4f after one tick, want the turn still under way", heading(dirs[0]))
	}
	if dirs[1] != east {
		t.Errorf("heading %v after the second tick, want east exactly", dirs[1])
	}
}

func TestSteering_StationaryEntityTakesTheHeadingWhole(t *testing.T) {
	var stationary geom.Vec
	if dirs := steerTicks(t, Steering{Want: east, TurnRate: 0.01}, stationary, 1); dirs[0] != east {
		t.Errorf("heading %v after one tick from a standstill, want east", dirs[0])
	}
}

func TestSteering_LeavesHeadingAloneWithNoRequest(t *testing.T) {
	if dirs := steerTicks(t, Steering{}, north, 2); dirs[0] != north || dirs[1] != north {
		t.Errorf("headings %v, want north throughout", dirs)
	}
}

// asking is a behavior that renews the same request every tick, the way any
// behavior watching a lasting stimulus does.
type asking struct {
	towards geom.Vec
	query   *goke.Query
	steer   goke.Comp[Steering]
}

func (a *asking) Init(si *goke.SysInit) { a.query = si.NewQueryBuilder(&a.steer).Build() }

func (a *asking) Update(*goke.CmdBuf, time.Duration) {
	a.query.All()
	for a.query.Next() {
		steers := a.steer.Slice(a.query.Cursor())
		for i := range steers {
			steers[i].Request(a.towards)
		}
	}
}

func TestSteering_LastingStimulusStillTurnsTheEntity(t *testing.T) {
	wm := testWorld()
	wm.RegisterBehavior(&asking{towards: north})
	wm.populate(testKind(
		Position{AABB: plane.NewAABB(geom.NewVec(500, 500), 10, 10)},
		Velocity{Dir: east, Value: 1},
		kind.Const(Steering{Reflex: 3, TurnRate: 0.12}),
	), []any{nil})

	var base goke.Comp[Base]
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&base).Build()
	}})...)
	wm.RegSystems(ecs)
	ecs.SetPlan(wm.RunPlan)
	for range 30 {
		ecs.Tick(time.Second / 60)
	}

	query.All()
	for query.Next() {
		for _, b := range base.Slice(query.Cursor()) {
			if b.Vel.Dir != north {
				t.Errorf("heading %v after 30 ticks of being asked north, want north — a quarter turn takes 14", b.Vel.Dir)
			}
		}
	}
}

func TestSteering_KeepsActingOnTheLastDecisionWhileReacting(t *testing.T) {
	const rate = 0.1
	dirs := steerTicks(t, Steering{Want: east, Pending: north, Reflex: 3, Delay: 3, TurnRate: rate}, north, 2)

	for i, d := range dirs {
		if want := heading(north) - rate*float64(i+1); math.Abs(heading(d)-want) > 1e-9 {
			t.Errorf("tick %d heading %.4f, want %.4f — still swinging east while north waits its turn", i+1, heading(d), want)
		}
	}
}
