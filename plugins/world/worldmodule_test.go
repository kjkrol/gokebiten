package world_test

import (
	"strings"
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

func testConfig() world.Config { return world.Config{Width: 1000, Height: 1000} }
func testPopulation() world.Population {
	return world.Population{MaxCount: 10, MinSize: 1, MaxSize: 100}
}

func testPos() world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](0, 0), 10, 10)}
}

type fakeSpawner struct{ pos world.Position }

func (f fakeSpawner) Spawn(index, count int) (world.Position, world.Velocity) {
	return f.pos, world.Velocity{}
}
func (f fakeSpawner) Components() []goke.Addable             { return nil }
func (f fakeSpawner) Init(*goke.Cursor, int, int, uid.UID64) {}

type fakePlacement struct{ pos world.Position }

func (f fakePlacement) Place(index, count int) world.Position  { return f.pos }
func (f fakePlacement) Components() []goke.Addable             { return nil }
func (f fakePlacement) Init(*goke.Cursor, int, int, uid.UID64) {}

type neitherSpawnerNorPlacement struct{}

func (neitherSpawnerNorPlacement) Components() []goke.Addable             { return nil }
func (neitherSpawnerNorPlacement) Init(*goke.Cursor, int, int, uid.UID64) {}

func TestModule_Populate_EndToEnd(t *testing.T) {
	wm := world.NewModule(testConfig(), testPopulation())

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

func TestModule_Populate_WithPlacement_EndToEnd(t *testing.T) {
	wm := world.NewModule(testConfig(), testPopulation())

	wm.Populate(2, fakePlacement{pos: testPos()})

	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)

	if wm.Telemetry().Count != 2 {
		t.Errorf("Telemetry().Count = %d, want 2", wm.Telemetry().Count)
	}
}

func TestModule_Populate_WrongPopulatorType_Panics(t *testing.T) {
	wm := world.NewModule(testConfig(), testPopulation())
	wm.Populate(1, neitherSpawnerNorPlacement{})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic when populators[0] implements neither world.Spawner nor world.Placement")
		}
		if !strings.Contains(r.(string), "Spawner") || !strings.Contains(r.(string), "Placement") {
			t.Errorf("panic message = %q, want it to mention both Spawner and Placement", r)
		}
	}()
	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)
}

func TestModule_Populate_NoPopulators_Panics(t *testing.T) {
	wm := world.NewModule(testConfig(), testPopulation())
	wm.Populate(1)

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic when Populate has no populators at all")
		}
	}()
	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)
}
