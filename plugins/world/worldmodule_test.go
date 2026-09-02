package world_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

func testConfig() world.Config {
	return world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 1, MaxSize: 100},
	}
}

func testPos() world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](0, 0), 10, 10)}
}

type fakeSpawner struct{ pos world.Position }

func (f fakeSpawner) Spawn(index, count int) (world.Position, world.Velocity) {
	return f.pos, world.Velocity{}
}

func TestModule_Populate_EndToEnd(t *testing.T) {
	wm := world.NewModule(testConfig())

	wm.Populate(3, fakeSpawner{pos: testPos()})

	ecs := goke.New()
	var pos goke.Comp[world.Position]
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
			if p.TopLeft != testPos().TopLeft {
				t.Errorf("spawned entity position = %+v, want %+v", p.TopLeft, testPos().TopLeft)
			}
		}
	}
	if count != 3 {
		t.Errorf("found %d entities with Position, want 3", count)
	}
}
