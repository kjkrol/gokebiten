package main

import (
	"testing"

	"github.com/kjkrol/gram/plugins/board"
)

func TestIslandLayout_HasEveryKindAndARoadBetweenTheStops(t *testing.T) {
	grid := board.DefaultGrids{}.Square(GridWidth, GridHeight, CellSize)
	layout, stops := islandLayout(grid)
	if layout.Default != "water" {
		t.Errorf("default kind %q, want water round the island", layout.Default)
	}
	count := map[string]int{}
	kinds := map[board.CellID]string{}
	for _, e := range layout.Cells {
		count[e.Kind]++
		kinds[e.Cell] = e.Kind
	}
	for _, k := range []string{"field", "forest", "hills", "mountain", "road"} {
		if count[k] == 0 {
			t.Errorf("no %s cells", k)
		}
	}
	if len(stops) != UnitCount {
		t.Errorf("%d road stops, want %d", len(stops), UnitCount)
	}
	for _, s := range stops {
		if kinds[s] != "road" {
			t.Errorf("stop %v is %q, want road", s, kinds[s])
		}
	}
	if island := len(layout.Cells); island < GridWidth*GridHeight/3 || island > GridWidth*GridHeight*2/3 {
		t.Errorf("the island covers %d of %d cells, want a third to two thirds", island, GridWidth*GridHeight)
	}
}
