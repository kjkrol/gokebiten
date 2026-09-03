package board_test

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

func newSaveLoadTestWorldPlugin() *world.Plugin {
	return world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
}

// TestPlugin_SaveLoad_TerrainRoundTrip guards that board.Plugin's TerrainMap
// is saved/restored via Saveable, without the caller ever passing it to
// Persistence.Save/Load explicitly.
func TestPlugin_SaveLoad_TerrainRoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"
	grid := board.NewSquareGrid(5, 5, 10)
	wall := board.CellKind{Name: "wall", Cost: 1, Passable: false}
	cell, ok := grid.CellAt(geom.NewVec(21.0, 21.0))
	if !ok {
		t.Fatal("expected (21,21) to land inside the 5x5 grid")
	}

	game := gokebiten.NewGame(&gokebiten.GameProps{})
	if err := game.UsePlugin(newSaveLoadTestWorldPlugin()); err != nil {
		t.Fatalf("UsePlugin(world): %v", err)
	}
	boardPlugin := board.NewPlugin(grid, &board.SingleOccupancy{}, nil)
	if err := game.UsePlugin(boardPlugin); err != nil {
		t.Fatalf("UsePlugin(board): %v", err)
	}
	game.Resources().Get[*board.Board]().Set(cell, wall)

	if err := game.Persistence.Save(basePath, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	game2 := gokebiten.NewGame(&gokebiten.GameProps{})
	if err := game2.UsePlugin(newSaveLoadTestWorldPlugin()); err != nil {
		t.Fatalf("UsePlugin(world) 2: %v", err)
	}
	boardPlugin2 := board.NewPlugin(grid, &board.SingleOccupancy{}, nil)
	if err := game2.UsePlugin(boardPlugin2); err != nil {
		t.Fatalf("UsePlugin(board) 2: %v", err)
	}

	if err := game2.Persistence.Load(basePath, ""); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := game2.Resources().Get[*board.Board]().Kind(cell); got != wall {
		t.Errorf("Board().Kind(cell) after Load = %+v, want %+v", got, wall)
	}
}
