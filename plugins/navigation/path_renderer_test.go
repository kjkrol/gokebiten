package navigation_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/navigation"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

func TestPathPoints_NoPathYet_StraightToTarget(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	start, _ := grid.CellIndex(0, 0)
	target, _ := grid.CellIndex(4, 0)
	pos := positionAt(grid, start)

	points := navigation.PathPoints(grid, pos, navigation.MoveOrder{Target: target})

	want := []geom.Vec[float64]{board.Center(pos), grid.CellCenter(target)}
	assertPoints(t, points, want)
}

func TestPathPoints_PartiallyConsumedPath_SkipsPassedSteps(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	start, _ := grid.CellIndex(0, 0)
	c1, _ := grid.CellIndex(1, 0)
	c2, _ := grid.CellIndex(2, 0)
	target, _ := grid.CellIndex(3, 0)
	pos := positionAt(grid, start)

	var p navigation.Path
	p.Steps[0] = c1
	p.Steps[1] = c2
	p.Steps[2] = target
	p.Length = 3
	p.Index = 1 // c1 already consumed

	points := navigation.PathPoints(grid, pos, navigation.MoveOrder{Target: target, Path: p})

	want := []geom.Vec[float64]{board.Center(pos), grid.CellCenter(c2), grid.CellCenter(target)}
	assertPoints(t, points, want)
}

func TestPathPoints_LastPointAlwaysTarget(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	start, _ := grid.CellIndex(0, 0)
	target, _ := grid.CellIndex(2, 0)
	pos := positionAt(grid, start)

	var p navigation.Path
	p.Steps[0] = target
	p.Length = 1
	p.Index = 0

	points := navigation.PathPoints(grid, pos, navigation.MoveOrder{Target: target, Path: p})

	last := points[len(points)-1]
	wantLast := grid.CellCenter(target)
	if last != wantLast {
		t.Errorf("last point = %v, want %v (CellCenter(target))", last, wantLast)
	}
}

func positionAt(grid *board.SquareGrid, c board.CellID) world.Position {
	return world.Position{AABB: board.CellAABB(grid, c, 8)}
}

func assertPoints(t *testing.T, got, want []geom.Vec[float64]) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("pathPoints returned %d points, want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("pathPoints[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
