package world

import (
	"testing"

	"github.com/kjkrol/goke/v3"
)

func TestWorld_Populate_EndToEnd(t *testing.T) {
	wm := testWorld()

	spawner := NewSpawner(
		func(index, count int) Position { return spawnerTestPos() },
		func(index int) Velocity { return Velocity{} },
	)
	wm.Populate(3, spawner)

	ecs := goke.New()
	var pos goke.Comp[Position]
	var q *goke.Query
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&pos).Build()
	}})
	ecs.Setup(systems...)

	if wm.Telemetry().Count != 3 {
		t.Errorf("Telemetry().Count = %d, want 3", wm.Telemetry().Count)
	}

	count := 0
	q.All()
	for q.Next() {
		for _, p := range pos.Slice(q.Cursor()) {
			count++
			if p.TopLeft != spawnerTestPos().TopLeft {
				t.Errorf("spawned entity position = %+v, want %+v", p.TopLeft, spawnerTestPos().TopLeft)
			}
		}
	}
	if count != 3 {
		t.Errorf("found %d entities with Position, want 3", count)
	}
}

func TestWorld_RegSystems_IsIdempotent(t *testing.T) {
	wm := testWorld()
	ecs := goke.New()

	// Must not panic or double-register systems when called more than once.
	wm.RegSystems(ecs)
	wm.RegSystems(ecs)
}
