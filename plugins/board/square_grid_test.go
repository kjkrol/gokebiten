package board_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokg/geom"
)

func TestSquareGrid_NonToroidal_EdgeExcludesNeighbors(t *testing.T) {
	g := board.NewSquareGrid(4, 4, 10)
	corner := g.Neighbors(cellAtXY(g, 0, 0))
	if len(corner) != 2 {
		t.Errorf("corner cell has %d neighbors, want 2 (no wrap)", len(corner))
	}
	if !g.Contains(cellAtXY(g, 3, 3)) {
		t.Error("expected the last cell to be contained")
	}
}

func TestSquareGrid_Toroidal_EdgeWrapsNeighborsAndDistance(t *testing.T) {
	g := &board.SquareGrid{Width: 4, Height: 4, CellSize: 10, Toroidal: true}

	first, last := cellAtXY(g, 0, 0), cellAtXY(g, 3, 0)
	neighbors := g.Neighbors(first)
	if len(neighbors) != 4 {
		t.Fatalf("toroidal corner has %d neighbors, want 4", len(neighbors))
	}
	found := false
	for _, n := range neighbors {
		if n == last {
			found = true
		}
	}
	if !found {
		t.Error("expected column 0 and column Width-1 to be neighbors on a toroidal grid")
	}

	if d := g.Distance(first, last); d != 1 {
		t.Errorf("Distance(col 0, col Width-1) = %v, want 1 (wrap-around, not %d)", d, g.Width-1)
	}

	if !g.Contains(board.CellID(999999)) {
		t.Error("expected Contains to always be true on a toroidal grid")
	}
}

func TestSquareGrid_Toroidal_CellAtWrapsNegativePositions(t *testing.T) {
	g := &board.SquareGrid{Width: 4, Height: 4, CellSize: 10, Toroidal: true}
	c, ok := g.CellAt(geom.NewVec(-5.0, -5.0))
	if !ok {
		t.Fatal("expected a negative position to wrap to a valid cell")
	}
	if c != cellAtXY(g, 3, 3) {
		t.Errorf("CellAt(-5,-5) = %v, want the wrapped last cell", c)
	}
}

func cellAtXY(g *board.SquareGrid, x, y uint32) board.CellID {
	c, _ := g.CellAt(geom.NewVec(float64(x)*float64(g.CellSize)+1, float64(y)*float64(g.CellSize)+1))
	return c
}
