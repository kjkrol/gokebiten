package board_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

func TestValueExtras_WithEffect_EntersOccupancyOnSpawn(t *testing.T) {
	sqGrid := board.NewSquareGrid(5, 5, 10)
	occupancy := &board.SingleOccupancy{}
	target := cellAtXY(sqGrid, 2, 2)

	cfg := world.Config{
		Space:    world.SpaceCfg{Width: 50, Height: 50},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 8, MaxSize: 8},
	}
	wm := world.NewModule(cfg)
	placement := world.NewGridPlacement(50, 50, 8)
	wm.Populate(1,
		world.SpawnerFunc(func(index, count int) (world.Position, world.Velocity) {
			return placement.Place(index, count), world.Velocity{}
		}),
		world.NewValueExtras(func(index int) board.Cell {
			return board.Cell{ID: target}
		}).WithEffect(func(c board.Cell, id uid.UID64) {
			occupancy.Enter(c.ID, id)
		}),
	)

	ecs := goke.New()
	var cell goke.Comp[board.Cell]
	var q *goke.Query
	systems := append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
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
