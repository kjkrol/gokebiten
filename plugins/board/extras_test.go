package board_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

func TestValueExtras_WithEffect_EntersOccupancyOnSpawn(t *testing.T) {
	sqGrid := board.DefaultGrids{}.Square(5, 5, 10)
	occupancy := &board.SingleOccupancy{}
	target, _ := sqGrid.CellIndex(2, 2)

	cfg := world.Config{
		Space:    world.SpaceCfg{Width: 50, Height: 50},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 8, MaxSize: 8},
	}
	plugin := world.NewPlugin(cfg)
	placement := world.NewGridPlacement(50, 50, 8)
	spawner := world.NewSpawner(
		func(index, count int) world.Position { return placement.Place(index, count) },
		func(index int) world.Velocity { return world.Velocity{} },
	).WithEffect(func(index int) board.Cell {
		return board.Cell{ID: target}
	}, func(c board.Cell, id uid.UID64) {
		occupancy.Enter(c.ID, id)
	})
	plugin.Populate(1, spawner)

	ecs := goke.New()
	var pending []func() []goke.System
	ctx := plugins.NewGameCtx(plugins.NewResources(), ecs,
		func(any) {}, func(p func() []goke.System) { pending = append(pending, p) })
	if err := plugin.Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	var cell goke.Comp[board.Cell]
	var q *goke.Query
	var systems []goke.System
	for _, produce := range pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&cell).Build()
	}})
	ecs.Setup(systems...)

	q.All()
	found := false
	for q.Next() {
		cur := q.Cursor()
		cells := cell.Slice(cur)
		for i := range cur.IDs {
			found = true
			if cells[i].ID != target {
				t.Errorf("Cell.ID = %v, want %v", cells[i].ID, target)
			}
		}
	}
	if !found {
		t.Fatal("expected the spawned entity to exist")
	}
	if occupancy.CanEnter(target, uid.UID64(999)) {
		t.Error("expected occupancy.Enter to have claimed the target cell for the spawned entity")
	}
}
