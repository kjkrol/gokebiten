package bench_test

import (
	"testing"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
)

// mover is the row a moving 20x20 box spawns from.
type mover struct{ x, y float64 }

// benchWorld installs, on ctx, a world of n entities, 20x20 each on a 30-unit lattice, all
// drifting right at 60 units a second across a 4000x4000 torus, and returns its ECS ready to tick
// the world plugin alone.
func benchWorld(b *testing.B, ctx *headless, n int) *goke.ECS {
	b.Helper()
	w := ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: 4000, Height: 4000, Edges: aabbworld.Torus},
		Entities: world.EntitiesCfg{MaxCount: n, MinSize: 1, MaxSize: 100},
	})
	movers := kind.Define[mover](w.Kinds(), "mover", kind.Spec{
		kind.Load(func(m mover) world.Position {
			return world.Position{AABB: plane.NewAABB(geom.NewVec(m.x, m.y), 20, 20)}
		}),
		kind.Const(world.Velocity{Dir: geom.NewVec(1, 0), Value: 60}),
	})
	side := 1
	for side*side < n {
		side++
	}
	entries := make([]kind.Entry, 0, n)
	for i := range n {
		entries = append(entries, movers.Entry(mover{float64(10 + (i%side)*30), float64(10 + (i/side)*30)}))
	}
	w.Seed(entries...)
	return ctx.start(b, w.RunPlan)
}

// Benchmark_World_Tick is one tick of the world plugin alone: steering, velocity, movement under
// the edge rules, and the space rebuilt from every entity.
func Benchmark_World_Tick(b *testing.B) {
	for _, n := range []int{1000, 5000} {
		b.Run(entities(n), func(b *testing.B) {
			ecs := benchWorld(b, newHeadless(), n)
			b.ReportAllocs()
			for b.Loop() {
				ecs.Tick(step)
			}
		})
	}
}

// Benchmark_World_PositionScan reads every entity's Base through a goke query, chunk by chunk:
// the floor under any system that walks the population.
func Benchmark_World_PositionScan(b *testing.B) {
	for _, n := range []int{1000, 5000} {
		b.Run(entities(n), func(b *testing.B) {
			var base goke.Comp[world.Base]
			var query *goke.Query
			ctx := newHeadless()
			ctx.pending = append(ctx.pending, func() []goke.System {
				return []goke.System{goke.SystemFn{OnInit: func(si *goke.SysInit) { query = si.NewQueryBuilder(&base).Build() }}}
			})
			benchWorld(b, ctx, n)
			var sink float64
			b.ReportAllocs()
			for b.Loop() {
				query.All()
				for query.Next() {
					for _, e := range base.Slice(query.Cursor()) {
						sink += e.Pos.TopLeft.X
					}
				}
			}
			_ = sink
		})
	}
}
