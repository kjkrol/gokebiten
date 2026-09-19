package flee_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/vision/strategies/flee"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

type installCtx struct {
	ecs     *goke.ECS
	pending []func() []goke.System
}

func (c *installCtx) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(*goke.SysInit) { m.RegSystems(c.ecs) }}
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}
func (c *installCtx) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.pending = append(c.pending, p.SetupSystems)
	}
}
func (c *installCtx) RegSys(f func() goke.System) goke.Runnable { return c.ecs.RegSys(f()) }
func (c *installCtx) ECS() *goke.ECS                            { return c.ecs }

type body struct{ x, y float64 }

func at(d body) world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec(d.x, d.y), 10, 10)}
}

// run installs world+vision, registers flee, spawns a skittish observer at
// runner plus one threat per entry, and ticks once. It returns the observer's
// heading afterwards.
func run(t *testing.T, runner body, facing geom.Vec, threats ...body) geom.Vec {
	t.Helper()
	return runWith(t, nil, runner, facing, threats...)
}

func runWith(t *testing.T, tune func(*flee.Behavior), runner body, facing geom.Vec, threats ...body) geom.Vec {
	t.Helper()

	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 2000, Height: 2000},
		Entities: world.EntitiesCfg{MaxCount: 16, MinSize: 1, MaxSize: 100},
	})
	v := vision.NewPlugin(w)
	behavior := flee.New()
	if tune != nil {
		tune(behavior)
	}
	w.RegisterBehavior(behavior)

	ctx := &installCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		t.Fatalf("world Install: %v", err)
	}
	if err := v.Install(ctx); err != nil {
		t.Fatalf("vision Install: %v", err)
	}

	dict := w.EntKindDict()
	dict.Define("runner", func(k world.Kind[body]) world.EntKind {
		return world.EntKind{
			Position: k.Load(at),
			Velocity: k.Const(world.Velocity{Dir: facing, Value: 1}),
			Components: []world.ComponentTemplate{
				k.Const(vision.Sight{Facing: facing, HalfAngle: math.Pi / 2.5, Radius: 600}),
				k.Const(vision.Sighted{}),
				k.Const(world.Steering{}),
				k.Const(flee.Skittish{}),
			},
		}
	})
	dict.Define("threat", func(k world.Kind[body]) world.EntKind {
		return world.EntKind{Position: k.Load(at), Velocity: k.Const(world.Velocity{})}
	})

	w.Seed(dict.Entry("runner", runner))
	for _, th := range threats {
		w.Seed(dict.Entry("threat", th))
	}
	if err := w.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	var vel goke.Comp[world.Velocity]
	var query *goke.Query
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&vel).Include(goke.Include[flee.Skittish]()).Build()
	}})
	ctx.ecs.Setup(systems...)

	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) { v.RunPlan(rc, d); w.RunPlan(rc, d) })
	ctx.ecs.Tick(time.Second / 60)

	var out geom.Vec
	query.All()
	for query.Next() {
		for _, got := range vel.Slice(query.Cursor()) {
			out = got.Dir
		}
	}
	return out
}

func heading(v geom.Vec) float64 { return math.Atan2(v.Y, v.X) }

// One threat dead ahead: the runner must end up heading away from it.
func TestFlee_TurnsAwayFromWhatItSees(t *testing.T) {
	east := geom.NewVec(1.0, 0.0)
	got := run(t, body{x: 500, y: 500}, east, body{x: 800, y: 500})

	if math.Abs(heading(got)-math.Pi) > 1e-6 {
		t.Errorf("heading %.4f rad (%v), want pi — straight away from the threat", heading(got), got)
	}
}

// Nothing in view, nothing to decide: the heading is left alone.
func TestFlee_LeavesTheHeadingAloneWithNothingInSight(t *testing.T) {
	east := geom.NewVec(1.0, 0.0)
	if got := run(t, body{x: 500, y: 500}, east); got != east {
		t.Errorf("heading %v with an empty cone, want it untouched (%v)", got, east)
	}
}

// Two threats either side push the runner down the middle between them, not at
// whichever happened to be reported first.
func TestFlee_CombinesEveryThreatIntoOneTurn(t *testing.T) {
	east := geom.NewVec(1.0, 0.0)
	got := run(t, body{x: 500, y: 500}, east,
		body{x: 800, y: 300},
		body{x: 800, y: 700},
	)

	if math.Abs(heading(got)-math.Pi) > 0.05 {
		t.Errorf("heading %.4f rad, want about pi — the two pushes should cancel sideways", heading(got))
	}
}

// Switched off, the behavior leaves the heading alone even with something in
// plain view — the demo toggles this to show what the avoidance is worth.
func TestFlee_SwitchedOffLeavesTheHeadingAlone(t *testing.T) {
	east := geom.NewVec(1.0, 0.0)
	got := runWith(t, func(b *flee.Behavior) { b.SetEnabled(false) },
		body{x: 500, y: 500}, east, body{x: 800, y: 500})

	if got != east {
		t.Errorf("heading %v with the behavior off, want it untouched (%v)", got, east)
	}
}
