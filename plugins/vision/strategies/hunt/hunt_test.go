package hunt_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/vision/strategies/hunt"
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

// run is search with the looking around switched off — a pure chaser.
func run(t *testing.T, hunter body, prey []body, bystanders []body) geom.Vec {
	t.Helper()
	return search(t, 0, hunter, prey, bystanders)
}

// search installs world+vision, registers hunt, spawns a predator at hunter
// facing east plus the given prey and bystanders, and ticks once. It returns
// the predator's heading afterwards.
func search(t *testing.T, lookEvery time.Duration, hunter body, prey []body, bystanders []body) geom.Vec {
	t.Helper()
	east := geom.NewVec(1.0, 0.0)

	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 2000, Height: 2000},
		Entities: world.EntitiesCfg{MaxCount: 16, MinSize: 1, MaxSize: 100},
	})
	v := vision.NewPlugin(w)
	if err := v.RegisterBehavior(plugin.Between[hunt.Predator, hunt.Prey](hunt.Chase(lookEvery))); err != nil {
		t.Fatalf("RegisterBehavior: %v", err)
	}

	ctx := &installCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		t.Fatalf("world Install: %v", err)
	}
	if err := v.Install(ctx); err != nil {
		t.Fatalf("vision Install: %v", err)
	}

	dict := w.EntKindDict()
	dict.Define("hunter", func(k world.Kind[body]) world.EntKind {
		return world.EntKind{
			Position: k.Load(at),
			Velocity: k.Const(world.Velocity{Dir: east, Value: 1}),
			Components: []world.ComponentTemplate{
				k.Const(vision.Sight{Facing: east, HalfAngle: math.Pi / 2.5, Radius: 600}),
				k.Const(world.Steering{}),
				k.Const(hunt.Predator{}),
			},
		}
	})
	dict.Define("prey", func(k world.Kind[body]) world.EntKind {
		return world.EntKind{
			Position:   k.Load(at),
			Velocity:   k.Const(world.Velocity{}),
			Components: []world.ComponentTemplate{k.Const(hunt.Prey{})},
		}
	})
	dict.Define("bystander", func(k world.Kind[body]) world.EntKind {
		return world.EntKind{Position: k.Load(at), Velocity: k.Const(world.Velocity{})}
	})

	w.Seed(dict.Entry("hunter", hunter))
	for _, p := range prey {
		w.Seed(dict.Entry("prey", p))
	}
	for _, b := range bystanders {
		w.Seed(dict.Entry("bystander", b))
	}
	if err := w.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	var base goke.Comp[world.Base]
	var query *goke.Query
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&base).Include(goke.Include[hunt.Predator]()).Build()
	}})
	ctx.ecs.Setup(systems...)

	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) { v.RunPlan(rc, d); w.RunPlan(rc, d) })
	ctx.ecs.Tick(time.Second / 60)

	var out geom.Vec
	query.All()
	for query.Next() {
		for _, got := range base.Slice(query.Cursor()) {
			out = got.Vel.Dir
		}
	}
	return out
}

func heading(v geom.Vec) float64 { return math.Atan2(v.Y, v.X) }

func TestHunt_TurnsTowardsThePreyItSees(t *testing.T) {
	got := run(t, body{x: 500, y: 500}, []body{{x: 800, y: 800}}, nil)

	if want := math.Pi / 4; math.Abs(heading(got)-want) > 1e-6 {
		t.Errorf("heading %.4f rad (%v), want %.4f — straight at the prey", heading(got), got, want)
	}
}

// Sighted is ordered by distance, so the first prey in it is the nearest one —
// and the far one must not pull the chase off course.
func TestHunt_GoesForTheNearestPrey(t *testing.T) {
	got := run(t, body{x: 500, y: 500}, []body{{x: 900, y: 500}, {x: 600, y: 500}}, nil)

	if math.Abs(heading(got)) > 1e-6 {
		t.Errorf("heading %.4f rad, want 0 — at the near prey, dead east", heading(got))
	}
}

// Seeing something is not the same as it being prey: Query.Seek finds any
// entity, so a bystander in full view must leave the course alone.
func TestHunt_IgnoresWhatIsNotPrey(t *testing.T) {
	east := geom.NewVec(1.0, 0.0)

	if got := run(t, body{x: 500, y: 500}, nil, []body{{x: 800, y: 800}}); got != east {
		t.Errorf("heading %v, want it untouched (%v) — a bystander is not prey", got, east)
	}
}

// With nobody in view the predator looks around: a quarter turn, to whichever
// side the coin falls — so what is pinned down is the right angle, not the side.
func TestHunt_LooksAroundWhenThereIsNobodyToChase(t *testing.T) {
	got := search(t, time.Millisecond, body{x: 500, y: 500}, nil, nil)

	if math.Abs(got.X) > 1e-9 || math.Abs(math.Abs(got.Y)-1) > 1e-9 {
		t.Errorf("heading %v, want a quarter turn off east — straight up or straight down", got)
	}
}

// Between looks it runs straight: the search is a look, a stretch, a look —
// not a spin on the spot.
func TestHunt_RunsStraightBetweenLooks(t *testing.T) {
	east := geom.NewVec(1.0, 0.0)

	if got := search(t, time.Hour, body{x: 500, y: 500}, nil, nil); got != east {
		t.Errorf("heading %v one tick into an hour's stretch, want it untouched (%v)", got, east)
	}
}

// Prey in view always wins over the search, however overdue a look is.
func TestHunt_ChasesRatherThanLooksAround(t *testing.T) {
	got := search(t, time.Millisecond, body{x: 500, y: 500}, []body{{x: 800, y: 800}}, nil)

	if want := math.Pi / 4; math.Abs(heading(got)-want) > 1e-6 {
		t.Errorf("heading %.4f rad, want %.4f — at the prey, not off to one side", heading(got), want)
	}
}
