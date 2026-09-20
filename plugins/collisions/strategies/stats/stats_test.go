package stats_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/stats"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	gokgspatial "github.com/kjkrol/gokg/spatial"
)

// overlapping runs the real collision engine, with s counting, over two
// detect-only boxes that overlap and therefore stay in contact tick after tick.
func overlapping(t *testing.T, s *stats.Stats, ticks int) {
	t.Helper()
	space, err := gokg.NewSpace(gokg.Config{
		Width: 1000, Height: 1000,
		BucketSize: gokgspatial.ResolutionFrom(64), BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("gokg.NewSpace: %v", err)
	}

	ecs := goke.New()
	engine := collisions.New(space, ecs, 10)
	if err := engine.RegisterBehavior(plugin.Between[plugin.Anything, plugin.Anything](stats.Count(s))); err != nil {
		t.Fatalf("RegisterBehavior: %v", err)
	}

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var base goke.Comp[world.Base]
		var coll goke.Comp[collisions.Collision]
		f := si.NewFactory(&base, &coll)
		f.Create(2)
		f.Next()
		for i, x := range []float64{100, 105} {
			box := plane.NewAABB(geom.NewVec(x, 100), 10, 10)
			base.Slice(&f.Cursor)[i].Pos = world.Position{AABB: box}
			space.Insert(f.IDs[i], box)
			space.SetCapabilities(f.IDs[i], collisions.CanCollide)
		}
		space.Flush(nil)
	}})
	engine.RegSystems(ecs)
	ecs.SetPlan(engine.RunPlan)
	for range ticks {
		ecs.Tick(time.Millisecond)
	}
}

// Both sides of a contact carry "anything", so the pair would match either way
// round — and must still count as the one contact it is.
func TestBehavior_CountsEachContactOnce(t *testing.T) {
	var s stats.Stats

	overlapping(t, &s, 1)

	if s.Counter != 1 {
		t.Errorf("Counter = %d, want 1 for one pair in contact", s.Counter)
	}
}

func TestBehavior_AccumulatesAcrossTicks(t *testing.T) {
	s := stats.Stats{Counter: 5}

	overlapping(t, &s, 3)

	if s.Counter != 8 {
		t.Errorf("Counter = %d, want 8 — three more ticks in contact on top of the 5 it started with", s.Counter)
	}
}
