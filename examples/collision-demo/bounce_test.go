package main

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// The demo is entities bouncing off each other, and that takes both halves:
// the collision engine publishing contacts, and a kind carrying what the
// bounce behavior is scoped to. Leave out either and the engine still runs,
// the build still passes, and the entities merely shove each other into
// clumps — so this ticks the real Stage and insists that courses change.
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
	// Nothing else in this demo writes Velocity, so without a bounce this is
	// exactly zero however long it runs.
	if changed*100 < len(before) {
		t.Errorf("%d of %d entities changed course over 60 ticks, want at least 1%% of them bouncing",
			changed, len(before))
	}
}

// velocities is a read-only view of every entity's Velocity, registered after
// the Stage's own single Setup — which is why it goes in through RegSys.
func velocities(ecs *goke.ECS) (*goke.Query, *goke.Comp[world.Velocity]) {
	vel := new(goke.Comp[world.Velocity])
	var q *goke.Query
	ecs.RegSys(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(vel).Build()
	}})
	return q, vel
}

// headings is every entity's current direction of travel, by id.
func headings(q *goke.Query, vel *goke.Comp[world.Velocity]) map[uid.UID64]geom.Vec {
	found := map[uid.UID64]geom.Vec{}
	q.All()
	for q.Next() {
		cursor := q.Cursor()
		velocities := vel.Slice(cursor)
		for i, id := range cursor.IDs {
			found[id] = velocities[i].Dir
		}
	}
	return found
}
