package world_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/plugins/world/spawners/grid"
	"github.com/kjkrol/uid"
)

type tag struct{}

type stat struct{ HP int }

func TestValueExtras_Init_WritesValue(t *testing.T) {
	wm := world.NewModule(testConfig())
	placement := grid.NewGridPlacement(50, 50, 8)
	wm.Populate(1,
		world.SpawnerFunc(func(index, count int) (world.Position, world.Velocity) {
			return placement.Place(index, count), world.Velocity{}
		}),
		world.NewValueExtras(func(index int) stat { return stat{HP: 7} }),
	)

	ecs := goke.New()
	var comp goke.Comp[stat]
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

func TestValueExtras_Init_ZeroSizeType_DoesNotPanic(t *testing.T) {
	wm := world.NewModule(testConfig())
	placement := grid.NewGridPlacement(50, 50, 8)
	wm.Populate(1,
		world.SpawnerFunc(func(index, count int) (world.Position, world.Velocity) {
			return placement.Place(index, count), world.Velocity{}
		}),
		world.NewValueExtras(func(index int) tag { return tag{} }),
	)

	ecs := goke.New()
	var q *goke.Query
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder().Include(goke.Include[tag]()).Build()
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

func TestValueExtras_WithEffect_RunsAfterWriteWithValueAndID(t *testing.T) {
	wm := world.NewModule(testConfig())
	placement := grid.NewGridPlacement(50, 50, 8)

	var gotHP int
	var gotID uid.UID64
	var calls int
	wm.Populate(1,
		world.SpawnerFunc(func(index, count int) (world.Position, world.Velocity) {
			return placement.Place(index, count), world.Velocity{}
		}),
		world.NewValueExtras(func(index int) stat { return stat{HP: 9} }).
			WithEffect(func(v stat, id uid.UID64) {
				calls++
				gotHP = v.HP
				gotID = id
			}),
	)

	ecs := goke.New()
	var comp goke.Comp[stat]
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
