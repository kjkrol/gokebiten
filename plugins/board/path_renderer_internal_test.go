package board

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

func TestPathPoints_NoPathYet_StraightToTarget(t *testing.T) {
	grid := NewSquareGrid(5, 1, 10)
	start := mustCell(t, grid, 0, 0)
	target := mustCell(t, grid, 4, 0)
	pos := positionAt(grid, start)

	points := pathPoints(grid, pos, Path{}, target)

	want := []geom.Vec[float64]{center(pos), grid.CellCenter(target)}
	assertPoints(t, points, want)
}

func TestPathPoints_PartiallyConsumedPath_SkipsPassedSteps(t *testing.T) {
	grid := NewSquareGrid(5, 1, 10)
	start := mustCell(t, grid, 0, 0)
	c1 := mustCell(t, grid, 1, 0)
	c2 := mustCell(t, grid, 2, 0)
	target := mustCell(t, grid, 3, 0)
	pos := positionAt(grid, start)

	var p Path
	p.Steps[0] = c1
	p.Steps[1] = c2
	p.Steps[2] = target
	p.Length = 3
	p.Index = 1 // c1 already consumed

	points := pathPoints(grid, pos, p, target)

	want := []geom.Vec[float64]{center(pos), grid.CellCenter(c2), grid.CellCenter(target)}
	assertPoints(t, points, want)
}

func TestPathPoints_LastPointAlwaysTarget(t *testing.T) {
	grid := NewSquareGrid(5, 1, 10)
	start := mustCell(t, grid, 0, 0)
	target := mustCell(t, grid, 2, 0)
	pos := positionAt(grid, start)

	var p Path
	p.Steps[0] = target
	p.Length = 1
	p.Index = 0

	points := pathPoints(grid, pos, p, target)

	last := points[len(points)-1]
	wantLast := grid.CellCenter(target)
	if last != wantLast {
		t.Errorf("last point = %v, want %v (CellCenter(target))", last, wantLast)
	}
}

func mustCell(t *testing.T, grid *SquareGrid, x, y uint32) CellID {
	t.Helper()
	c, ok := grid.CellAt(geom.NewVec(float64(x)*10+1, float64(y)*10+1))
	if !ok {
		t.Fatalf("CellAt(%d,%d) not found", x, y)
	}
	return c
}

func positionAt(grid *SquareGrid, c CellID) world.Position {
	return world.Position{AABB: CellAABB(grid, c, 8)}
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
