package main

import (
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

func TestStage_EntitiesBounceOffEachOther(t *testing.T) {
	ecs, stage := buildStage(t)
	q, vel := velocities(ecs)

	before := headings(q, vel)
	if len(before) != EntityCount {
		t.Fatalf("stage spawned %d entities, want %d", len(before), EntityCount)
	}
	for range 60 {
		ecs.Tick(benchStep)
	}
	after := headings(q, vel)

	if stage.collisionStats.Counter == 0 {
		t.Fatal("no contacts in 60 ticks — the entities are not meeting at all")
	}
	var changed int
	for id, was := range before {
		if now, ok := after[id]; ok && now != was {
			changed++
		}
	}
	if changed*100 < len(before) {
		t.Errorf("%d of %d entities changed course over 60 ticks, want at least 1%% of them bouncing",
			changed, len(before))
	}
}

// velocities is a read-only view of every entity's Velocity.
func velocities(ecs *goke.ECS) (*goke.Query, *goke.Comp[world.Base]) {
	vel := new(goke.Comp[world.Base])
	var q *goke.Query
	ecs.RegSys(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(vel).Build()
	}})
	return q, vel
}

// headings is every entity's current direction of travel, by id.
func headings(q *goke.Query, vel *goke.Comp[world.Base]) map[uid.UID64]geom.Vec {
	found := map[uid.UID64]geom.Vec{}
	q.All()
	for q.Next() {
		cursor := q.Cursor()
		velocities := vel.Slice(cursor)
		for i, id := range cursor.IDs {
			found[id] = velocities[i].Vel.Dir
		}
	}
	return found
}
