package board_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/uid"
)

// TestFindPath_UnreachableTarget_ReportsNotFound guards against a real bug
// in the astar dependency: when 'to' is unreachable, its Solver.Solve
// reconstructs a path to whatever node it last expanded instead of
// returning nil — so FindPath must itself verify the returned path actually
// ends at 'to' before reporting success.
func TestFindPath_UnreachableTarget_ReportsNotFound(t *testing.T) {
	grid := board.NewSquareGrid(5, 5, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}

	from := cellAtXY(grid, 0, 0)
	to := cellAtXY(grid, 2, 2)
	for _, n := range grid.Neighbors(to) {
		terrain.Set(n, board.CellProps{Cost: 1, Passable: false})
	}

	_, ok := board.FindPath(grid, terrain, occupancy, uid.UID64(1), from, to)
	if ok {
		t.Error("expected FindPath to report not-found for a target walled in on every side")
	}
}
