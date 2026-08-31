package board_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/board/grids"
	"github.com/kjkrol/uid"
)

// TestPathFinder_FindPath_UnreachableTarget_ReportsNotFound guards a target
// walled in on every side — regression coverage from the gokebiten side for
// a bug fixed upstream in astar v1.1.1 (Solver.Solve used to reconstruct a
// path to whatever node it last expanded instead of returning nil).
func TestPathFinder_FindPath_UnreachableTarget_ReportsNotFound(t *testing.T) {
	grid := grids.NewSquareGrid(5, 5, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}

	from := cellAtXY(grid, 0, 0)
	to := cellAtXY(grid, 2, 2)
	for _, n := range grid.Neighbors(to) {
		terrain.Set(n, board.CellProps{Cost: 1, Passable: false})
	}

	_, ok := board.NewPathFinder(grid).FindPath(terrain, occupancy, uid.UID64(1), from, to)
	if ok {
		t.Error("expected FindPath to report not-found for a target walled in on every side")
	}
}

// TestPathFinder_FindPath_ReusesSolverAcrossCalls guards that reusing one
// PathFinder for multiple, different queries doesn't leak state between
// calls — each call must succeed independently with a path ending at its
// own 'to'.
func TestPathFinder_FindPath_ReusesSolverAcrossCalls(t *testing.T) {
	grid := grids.NewSquareGrid(5, 5, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	entity := uid.UID64(1)

	pathFinder := board.NewPathFinder(grid)

	firstFrom, firstTo := cellAtXY(grid, 0, 0), cellAtXY(grid, 4, 0)
	firstPath, ok := pathFinder.FindPath(terrain, occupancy, entity, firstFrom, firstTo)
	if !ok {
		t.Fatal("expected the first query to find a path")
	}
	if firstPath.Length == 0 || firstPath.Steps[firstPath.Length-1] != firstTo {
		t.Errorf("first path = %+v, want it to end at %v", firstPath, firstTo)
	}

	secondFrom, secondTo := cellAtXY(grid, 0, 4), cellAtXY(grid, 4, 4)
	secondPath, ok := pathFinder.FindPath(terrain, occupancy, entity, secondFrom, secondTo)
	if !ok {
		t.Fatal("expected the second query, on the same reused PathFinder, to find a path")
	}
	if secondPath.Length == 0 || secondPath.Steps[secondPath.Length-1] != secondTo {
		t.Errorf("second path = %+v, want it to end at %v", secondPath, secondTo)
	}
}
