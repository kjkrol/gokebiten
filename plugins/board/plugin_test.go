package board

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestNewPlugin_SetsGridToroidalFromWorldConfig(t *testing.T) {
	grid := DefaultGrids{}.Square(5, 5, 10).(*squareGrid)
	if grid.Toroidal {
		t.Fatal("expected DefaultGrids.Square to default Toroidal=false")
	}

	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	NewPlugin(grid, &SingleOccupancy{}, worldPlugin)

	if !grid.Toroidal {
		t.Error("expected NewPlugin to set grid.Toroidal to match world.Config.Space.Toroidal=true")
	}
}

func newSeedTestPlugin(t *testing.T) (*Plugin, CellID) {
	t.Helper()
	grid := DefaultGrids{}.Square(5, 5, 10)
	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 50, Height: 50},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	p := NewPlugin(grid, &SingleOccupancy{}, worldPlugin)
	p.CellKindDict().Create(
		CellKind{Name: "grass", Cost: 1, Passable: true},
		CellKind{Name: "wall", Cost: 1, Passable: false},
	)
	cell, _ := grid.CellIndex(2, 2)
	return p, cell
}

func TestPlugin_SeedPopulate_AppliesLayout(t *testing.T) {
	p, wallCell := newSeedTestPlugin(t)
	other, _ := p.Res.Logic.Board.CellIndex(0, 0)
	p.Seed(Layout{Default: "grass", Cells: []CellEntry{{Kind: "wall", Cell: wallCell}}})

	if err := p.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}
	if got := p.Res.Logic.Board.Kind(wallCell).Name; got != "wall" {
		t.Errorf("Kind(wallCell) = %q, want %q", got, "wall")
	}
	if got := p.Res.Logic.Board.Kind(other).Name; got != "grass" {
		t.Errorf("Kind(other) = %q, want %q (Default)", got, "grass")
	}
}

func TestPlugin_Populate_UnknownKindChangesNothing(t *testing.T) {
	p, cell := newSeedTestPlugin(t)
	p.Seed(Layout{Default: "grass", Cells: []CellEntry{{Kind: "lava", Cell: cell}}})

	if err := p.Populate(); err == nil {
		t.Fatal("Populate: expected an error for unknown kind, got nil")
	}
	if got := p.Res.Logic.Board.Kind(cell).Name; got != "" {
		t.Errorf("Kind(cell) = %q, want untouched terrain", got)
	}
}
