package spatial_test

import (
	"strings"
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokebiten/spatial"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

func testConfig() spatial.Config { return spatial.Config{Width: 1000, Height: 1000} }
func testPopulation() spatial.Population {
	return spatial.Population{MaxCount: 10, MinSize: 1, MaxSize: 100}
}

func testPos() kinematics.Position {
	return kinematics.Position{AABB: plane.NewAABB(geom.NewVec[uint32](0, 0), 10, 10)}
}

type fakeSpawner struct{ pos kinematics.Position }

func (f fakeSpawner) Spawn(index, count int) (kinematics.Position, kinematics.Velocity) {
	return f.pos, kinematics.Velocity{}
}
func (f fakeSpawner) Components() []goke.Addable             { return nil }
func (f fakeSpawner) Init(*goke.Cursor, int, int, uid.UID64) {}

type fakePlacement struct{ pos kinematics.Position }

func (f fakePlacement) Place(index, count int) kinematics.Position { return f.pos }
func (f fakePlacement) Components() []goke.Addable                 { return nil }
func (f fakePlacement) Init(*goke.Cursor, int, int, uid.UID64)     {}

type neitherSpawnerNorPlacement struct{}

func (neitherSpawnerNorPlacement) Components() []goke.Addable             { return nil }
func (neitherSpawnerNorPlacement) Init(*goke.Cursor, int, int, uid.UID64) {}

func TestWorldModule_Populate_EndToEnd(t *testing.T) {
	wm := spatial.NewWorldModule(testConfig(), testPopulation())
	var tel kinematics.Telemetry

	wm.Populate(3, &tel, fakeSpawner{pos: testPos()})

	ecs := goke.New()
	var pos goke.Comp[kinematics.Position]
	var q *goke.Query
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&pos).Build()
	}})
	ecs.Setup(systems...)

	if tel.DynamicCount != 3 {
		t.Errorf("Telemetry.DynamicCount = %d, want 3", tel.DynamicCount)
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

func TestWorldModule_PopulateStatic_EndToEnd(t *testing.T) {
	wm := spatial.NewWorldModule(testConfig(), testPopulation())
	var tel kinematics.Telemetry

	wm.PopulateStatic(2, &tel, fakePlacement{pos: testPos()})

	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)

	if tel.StaticCount != 2 {
		t.Errorf("Telemetry.StaticCount = %d, want 2", tel.StaticCount)
	}
}

func TestWorldModule_Populate_WrongPopulatorType_Panics(t *testing.T) {
	wm := spatial.NewWorldModule(testConfig(), testPopulation())
	var tel kinematics.Telemetry
	wm.Populate(1, &tel, neitherSpawnerNorPlacement{})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic when populators[0] doesn't implement kinematics.Spawner")
		}
		if !strings.Contains(r.(string), "Spawner") {
			t.Errorf("panic message = %q, want it to mention Spawner", r)
		}
	}()
	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)
}

func TestWorldModule_PopulateStatic_WrongPopulatorType_Panics(t *testing.T) {
	wm := spatial.NewWorldModule(testConfig(), testPopulation())
	var tel kinematics.Telemetry
	wm.PopulateStatic(1, &tel, neitherSpawnerNorPlacement{})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic when populators[0] doesn't implement kinematics.Placement")
		}
		if !strings.Contains(r.(string), "Placement") {
			t.Errorf("panic message = %q, want it to mention Placement", r)
		}
	}()
	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)
}

func TestWorldModule_Populate_NoPopulators_Panics(t *testing.T) {
	wm := spatial.NewWorldModule(testConfig(), testPopulation())
	var tel kinematics.Telemetry
	wm.Populate(1, &tel)

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic when Populate has no populators at all")
		}
	}()
	ecs := goke.New()
	ecs.Setup(wm.SetupSystems()...)
}
