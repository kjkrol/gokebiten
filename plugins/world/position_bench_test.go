package world

import (
	"github.com/kjkrol/aabbworld"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
)

// benchWorld spawns n entities and returns the module with an ECS wired to run its tick.
func benchWorld(b *testing.B, n int) (*goke.ECS, *module, *goke.Query, *goke.Comp[Base]) {
	b.Helper()

	wm := NewPlugin(Config{
		Space:    SpaceCfg{Width: 4000, Height: 4000, Edges: aabbworld.Torus},
		Entities: EntitiesCfg{MaxCount: n, MinSize: 1, MaxSize: 100},
	}).module

	side := 1
	for side*side < n {
		side++
	}
	for i := range n {
		pos := Position{AABB: plane.NewAABB(
			geom.NewVec(float64(10+(i%side)*30), float64(10+(i/side)*30)), 20, 20)}
		wm.populate(testKind(pos, Velocity{Dir: geom.NewVec(1.0, 0.0), Value: 60}), []any{nil})
	}

	base := new(goke.Comp[Base])
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(base).Build()
	}})...)
	wm.RegSystems(ecs)
	ecs.SetPlan(wm.RunPlan)
	return ecs, wm, query, base
}

func Benchmark_PositionScan(b *testing.B) {
	for _, n := range []int{1000, 5000} {
		b.Run(entityCount(n), func(b *testing.B) {
			_, _, query, base := benchWorld(b, n)
			var sink float64
			b.ReportAllocs()
			for b.Loop() {
				query.All()
				for query.Next() {
					cursor := query.Cursor()
					for _, e := range base.Slice(cursor) {
						sink += e.Pos.TopLeft.X
					}
				}
			}
			_ = sink
		})
	}
}

func Benchmark_WorldTick(b *testing.B) {
	for _, n := range []int{1000, 5000} {
		b.Run(entityCount(n), func(b *testing.B) {
			ecs, _, _, _ := benchWorld(b, n)
			b.ReportAllocs()
			for b.Loop() {
				ecs.Tick(time.Second / 60)
			}
		})
	}
}

func entityCount(n int) string {
	if n == 1000 {
		return "entities=1000"
	}
	return "entities=5000"
}
