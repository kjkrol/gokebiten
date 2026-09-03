package world_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

type spawnerTag struct{}

type spawnerStat struct{ HP int }

func TestSpawner_With_AttachesComponent(t *testing.T) {
	wm := world.NewWorld(testConfig())
	spawner := world.NewSpawner(
		func(index, count int) world.Position { return testPos() },
		func(index int) world.Velocity { return world.Velocity{} },
	).With(func(index int) spawnerStat { return spawnerStat{HP: 7} })
	wm.Populate(1, spawner)

	ecs := goke.New()
	var comp goke.Comp[spawnerStat]
	var q *goke.Query
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&comp).Build()
	}})
	ecs.Setup(systems...)

	q.All()
	found := false
	for q.Next() {
		cur := q.Cursor()
		vals := comp.Slice(cur)
		for i := range cur.IDs {
			found = true
			if vals[i].HP != 7 {
				t.Errorf("HP = %d, want 7", vals[i].HP)
			}
		}
	}
	if !found {
		t.Fatal("expected the spawned entity to exist")
	}
}

func TestSpawner_With_ZeroSizeType_DoesNotPanic(t *testing.T) {
	wm := world.NewWorld(testConfig())
	spawner := world.NewSpawner(
		func(index, count int) world.Position { return testPos() },
		func(index int) world.Velocity { return world.Velocity{} },
	).With(func(index int) spawnerTag { return spawnerTag{} })
	wm.Populate(1, spawner)

	ecs := goke.New()
	var q *goke.Query
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder().Include(goke.Include[spawnerTag]()).Build()
	}})
	ecs.Setup(systems...)

	q.All()
	found := false
	for q.Next() {
		found = true
	}
	if !found {
		t.Fatal("expected the spawned entity to be tagged")
	}
}

func TestSpawner_WithEffect_RunsAfterWriteWithValueAndID(t *testing.T) {
	wm := world.NewWorld(testConfig())

	var gotHP int
	var gotID uid.UID64
	var calls int
	spawner := world.NewSpawner(
		func(index, count int) world.Position { return testPos() },
		func(index int) world.Velocity { return world.Velocity{} },
	).WithEffect(func(index int) spawnerStat { return spawnerStat{HP: 9} },
		func(v spawnerStat, id uid.UID64) {
			calls++
			gotHP = v.HP
			gotID = id
		})
	wm.Populate(1, spawner)

	ecs := goke.New()
	var comp goke.Comp[spawnerStat]
	var q *goke.Query
	var wantID uid.UID64
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&comp).Build()
	}})
	ecs.Setup(systems...)

	q.All()
	for q.Next() {
		cur := q.Cursor()
		for _, id := range cur.IDs {
			wantID = id
		}
	}

	if calls != 1 {
		t.Fatalf("effect called %d times, want 1", calls)
	}
	if gotHP != 9 {
		t.Errorf("effect saw HP = %d, want 9", gotHP)
	}
	if gotID != wantID {
		t.Errorf("effect saw id = %v, want %v (the spawned entity's own id)", gotID, wantID)
	}
}
