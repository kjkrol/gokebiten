package vision_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// benchScan ticks n observers, each scanning a cone over a world they share, so
// the number measures one full vision pass — not one Space.Scan in isolation.
func benchScan(b *testing.B, n int, outlines bool) {
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 4000, Height: 4000},
		Entities: world.EntitiesCfg{MaxCount: n, MinSize: 1, MaxSize: 100},
	})
	v := vision.NewPlugin(w)

	ctx := &installCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		b.Fatal(err)
	}
	if err := v.Install(ctx); err != nil {
		b.Fatal(err)
	}

	comps := []world.ComponentTemplate{
		world.Const(vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: math.Pi / 6, Radius: 200}),
	}
	if outlines {
		comps = append(comps, world.Const(vision.SightOutline{}))
	}
	dict := w.EntKindDict()
	dict.Define("watcher", func(k world.Kind[spawn]) world.EntKind {
		return world.EntKind{
			Position:   k.Load(func(d spawn) world.Position { return at(d.x, d.y) }),
			Velocity:   k.Const(world.Velocity{}),
			Components: comps,
		}
	})

	side := int(math.Ceil(math.Sqrt(float64(n))))
	entries := make([]world.Entry, 0, n)
	for i := range n {
		entries = append(entries, dict.Entry("watcher", spawn{
			x: float64(100 + (i%side)*120),
			y: float64(100 + (i/side)*120),
		}))
	}
	w.Seed(entries...)
	if err := w.Populate(); err != nil {
		b.Fatal(err)
	}

	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) { v.RunPlan(rc, d) })

	b.ReportAllocs()
	for b.Loop() {
		ctx.ecs.Tick(time.Second / 60)
	}
}

func Benchmark_Scan(b *testing.B) {
	for _, n := range []int{100, 500} {
		b.Run(observers(n), func(b *testing.B) { benchScan(b, n, false) })
	}
}

func Benchmark_ScanWithOutlines(b *testing.B) {
	for _, n := range []int{100, 500} {
		b.Run(observers(n), func(b *testing.B) { benchScan(b, n, true) })
	}
}

func observers(n int) string {
	if n == 100 {
		return "observers=100"
	}
	return "observers=500"
}
