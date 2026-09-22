package world

import (
	"testing"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/uid"
)

// leaving is a world with one 10x10 entity heading east, four ticks from wholly crossing the edge.
func leaving(t *testing.T, edges aabbworld.Edges) (*Plugin, *goke.ECS, *goke.Query) {
	t.Helper()
	p := NewPlugin(Config{
		Space:    SpaceCfg{Width: 1000, Height: 1000, Edges: edges},
		Entities: EntitiesCfg{MaxCount: 10, MinSize: 1, MaxSize: 100},
	})
	wm := p.module
	wm.populate(testKind(
		Position{AABB: plane.NewAABB(geom.NewVec(985, 500), 10, 10)},
		Velocity{Dir: geom.NewVec(1, 0), Value: 300},
	), []any{nil})

	var base goke.Comp[Base]
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&base).Build()
	}})...)
	wm.RegSystems(ecs)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		wm.RunPlan(ctx, d)
		ctx.Sync()
	})
	return p, ecs, query
}

// everywhere is how many indexed pieces a query over the whole world finds.
func everywhere(space *aabbworld.Space) int {
	return space.Query(geom.NewAABBAt(geom.NewVec(0, 0), 1000, 1000), aabbworld.AnyCapability, func(uid.UID64) {})
}

func TestExit_AnEntityLeavingByAnOpenEdgeIsDespawnedByDefault(t *testing.T) {
	p, ecs, query := leaving(t, aabbworld.OpenX)

	ecs.Tick(time.Second / 60)
	ecs.Tick(time.Second / 60)
	if got := len(living(query)); got != 1 {
		t.Fatalf("%d entities alive while part of it is still inside, want 1", got)
	}
	ecs.Tick(time.Second / 60)
	ecs.Tick(time.Second / 60)
	if got := len(living(query)); got != 0 {
		t.Errorf("%d entities alive once it has wholly left, want 0", got)
	}
	if got := everywhere(p.module.space); got != 0 {
		t.Errorf("the index still holds %d entities", got)
	}
}

func TestExit_OnExitHearsOfEachLeaverOnceAndKeepsItAlive(t *testing.T) {
	p, ecs, query := leaving(t, aabbworld.OpenX)
	var heard []uid.UID64
	p.OnExit(func(_ plugin.Tick, id uid.UID64) { heard = append(heard, id) })

	for range 6 {
		ecs.Tick(time.Second / 60)
	}
	alive := living(query)
	if len(alive) != 1 {
		t.Fatalf("%d entities alive, want the leaver kept for OnExit to deal with", len(alive))
	}
	if len(heard) != 1 || !alive[heard[0]] {
		t.Errorf("OnExit heard %v over six ticks, want the leaver exactly once", heard)
	}
}

func TestExit_AClosedEdgeStopsTheEntityWhole(t *testing.T) {
	p, ecs, query := leaving(t, 0)
	p.OnExit(func(plugin.Tick, uid.UID64) { t.Error("OnExit called in a closed world") })

	for range 6 {
		ecs.Tick(time.Second / 60)
	}
	if got := len(living(query)); got != 1 {
		t.Fatalf("%d entities alive, want 1", got)
	}
	if got := everywhere(p.module.space); got != 1 {
		t.Errorf("the index holds %d entities, want the one resting against the edge", got)
	}
}
