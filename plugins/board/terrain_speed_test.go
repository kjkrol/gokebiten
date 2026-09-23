package board_test

import (
	"testing"

	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/world"
)

func TestTerrainSpeedModifier_ScalesByOneOverCost(t *testing.T) {
	grid := board.DefaultGrids{}.Square(3, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	at := func(x uint32) board.CellID { c, _ := grid.CellIndex(x, 0); return c }
	terrain.Set(at(0), board.CellKind{Cost: 2, Passable: true})   // slow
	terrain.Set(at(1), board.CellKind{Cost: 0.5, Passable: true}) // a boost, if a game wants one
	terrain.Set(at(2), board.CellKind{Cost: 3, Passable: false})  // never entered: no effect
	m := board.NewTerrainSpeedModifier(grid, terrain)

	for x, want := range map[uint32]float64{0: 0.5, 1: 2, 2: 1} {
		base := &world.Base{Pos: world.Position{AABB: board.CellAABB(grid, at(x), 4)}}
		if got := m.Apply(nil, 0, base, 1); got != want {
			t.Errorf("cell %d: factor %v, want %v", x, got, want)
		}
	}
}
