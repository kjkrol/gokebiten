package world

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

// despawnWorld spawns n entities at the test position, indexed in the space the
// way a real spawn leaves them, and returns their ids with a live Position query.
func despawnWorld(t *testing.T, wm *module, n int) ([]uid.UID64, *goke.ECS, *goke.Query, *goke.Comp[Base]) {
	t.Helper()
	wm.populate(EntKind{Position: Const(spawnerTestPos()), Velocity: Const(Velocity{})}, make([]any, n))

	base := new(goke.Comp[Base])
	var q *goke.Query
	var ids []uid.UID64
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(base).Build()
		q.All()
		for q.Next() {
			cursor := q.Cursor()
			bases := base.Slice(cursor)
			for i, id := range cursor.IDs {
				ids = append(ids, id)
				wm.space.Insert(id, bases[i].Pos.AABB)
			}
		}
		wm.space.Flush(nil)
	}})...)
	return ids, ecs, q, base
}

// run ticks ecs once with act as its whole plan — despawning needs a CmdBuf,
// and a CmdBuf only exists inside a system.
func run(ecs *goke.ECS, act func(*goke.CmdBuf)) {
	handle := ecs.RegSys(goke.SystemFn{OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) { act(cb) }})
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)
}

func living(q *goke.Query) map[uid.UID64]bool {
	found := map[uid.UID64]bool{}
	q.All()
	for q.Next() {
		for _, id := range q.Cursor().IDs {
			found[id] = true
		}
	}
	return found
}

// indexed reports which of the space's entities a probe over the whole world still finds.
func indexed(space *gokg.Space) map[uid.UID64]bool {
	box := spawnerTestPos().AABB
	found := map[uid.UID64]bool{}
	space.Neighbours(&box, 100, gokg.Plain, func(id uid.UID64, _ plane.FragPosition) {
		found[id] = true
	})
	return found
}

// A despawned entity has to leave both halves of the world: the ECS it is
// queried from, and the index everything else probes — a ghost in the index is
// still seen, still collided with, and still chased.
func TestDespawn_TakesTheEntityOutOfBothTheECSAndTheIndex(t *testing.T) {
	wm := testWorld()
	ids, ecs, q, _ := despawnWorld(t, wm, 3)

	gone := ids[1]
	run(ecs, func(cb *goke.CmdBuf) { wm.despawn(cb, gone) })
	wm.space.Flush(nil)

	if alive := living(q); alive[gone] || len(alive) != 2 {
		t.Errorf("entities left = %v, want the two that were not despawned", alive)
	}
	if seen := indexed(wm.space); seen[gone] {
		t.Errorf("the spatial index still offers %v — a probe would keep finding a ghost", gone)
	}
	if wm.telemetry.Count != 2 {
		t.Errorf("telemetry.Count = %d, want 2", wm.telemetry.Count)
	}
}

// Two hunters reaching the same prey in one tick is an ordinary thing to
// happen, and it costs the world one entity, not two.
func TestDespawn_TwiceInATickCountsOnce(t *testing.T) {
	wm := testWorld()
	ids, ecs, _, _ := despawnWorld(t, wm, 3)

	run(ecs, func(cb *goke.CmdBuf) {
		wm.despawn(cb, ids[0])
		wm.despawn(cb, ids[0])
	})

	if wm.telemetry.Count != 2 {
		t.Errorf("telemetry.Count = %d, want 2 — the second ask was for an entity already gone", wm.telemetry.Count)
	}
}
