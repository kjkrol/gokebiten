package navigation

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/selection"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/uid"
)

// riverBoard is a 5x3 board with water across the middle row, except a ford at column 4.
func riverBoard() (board.Grid, *board.TerrainMap) {
	grid := board.DefaultGrids{}.Square(5, 3, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
	for x := uint32(0); x < 4; x++ {
		c, _ := grid.CellIndex(x, 1)
		terrain.Set(c, board.CellKind{Name: "water", Cost: 1, Allows: board.Water})
	}
	return grid, terrain
}

func TestFindPath_KeepsEachDomainToItsOwnGround(t *testing.T) {
	grid, terrain := riverBoard()
	pf := newPathFinder(grid, terrain, &board.MultipleOccupancy{})
	at := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }

	path, ok := pf.findPath(uid.UID64(1), board.Land, at(0, 0), at(0, 2))
	if !ok {
		t.Fatal("a land unit found no way round the river")
	}
	for _, step := range path.Steps[:path.Length] {
		if terrain.Kind(step).Name == "water" {
			t.Fatalf("a land unit's route crosses water at %v", step)
		}
	}
	if path.Length < 4 {
		t.Errorf("route of %d steps, want the detour through the ford", path.Length)
	}

	boat, ok := pf.findPath(uid.UID64(2), board.Water, at(0, 1), at(3, 1))
	if !ok || boat.Length != 3 {
		t.Fatalf("a boat along the river: ok=%v, %d steps, want 3 straight", ok, boat.Length)
	}
	if _, ok := pf.findPath(uid.UID64(2), board.Water, at(0, 1), at(0, 0)); ok {
		t.Error("a boat found a route onto grass")
	}

	amphibious := board.Land | board.Water
	if direct, ok := pf.findPath(uid.UID64(3), amphibious, at(0, 0), at(0, 2)); !ok || direct.Length != 2 {
		t.Errorf("an amphibious unit: ok=%v, %d steps, want 2 straight across", ok, direct.Length)
	}
}

func TestCommandSystem_Update_IgnoresATargetTheUnitsDomainMayNotEnter(t *testing.T) {
	grid := board.DefaultGrids{}.Square(10, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Allows: board.Land})
	start, _ := grid.CellIndex(0, 0)
	lake, _ := grid.CellIndex(8, 0)
	terrain.Set(lake, board.CellKind{Name: "water", Cost: 1, Allows: board.Water})

	cmdState := &Resources{}
	cmds := newMoveCommandSystem(newPathFinder(grid, terrain, &board.SingleOccupancy{}), cmdState)

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Base]
	var mover goke.Comp[board.Mover]
	var selected goke.Comp[selection.Selected]
	var order goke.OptComp[MoveOrder]
	var readQuery *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &mover, &selected)
		f.Create(1)
		f.Next()
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0].Pos = world.Position{AABB: board.CellAABB(grid, start, 8)}
		mover.Slice(&f.Cursor)[0] = board.Mover{Domain: board.Land}
		readQuery = si.NewQueryBuilder().Optional(&order).Build()
		cmds.Init(si)
	}})
	handle := ecs.RegSys(cmds)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})

	cmdState.Pending = &MoveCommand{Cell: lake}
	ecs.Tick(time.Second)

	for readQuery.All(); readQuery.Next(); {
		if order.Present(readQuery.Cursor()) {
			t.Fatal("a land unit was ordered onto water")
		}
	}
}
