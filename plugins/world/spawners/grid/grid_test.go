package grid_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world/spawners/grid"
)

func TestGridPlacement_Place_SquareCount(t *testing.T) {
	p := grid.NewGridPlacement(100, 100, 10)

	cases := []struct {
		index int
		wantX uint32
		wantY uint32
	}{
		{0, 20, 20},
		{1, 70, 20},
		{2, 20, 70},
		{3, 70, 70},
	}
	for _, c := range cases {
		pos := p.Place(c.index, 4)
		if pos.TopLeft.X != c.wantX || pos.TopLeft.Y != c.wantY {
			t.Errorf("Place(%d, 4) = (%d,%d), want (%d,%d)", c.index, pos.TopLeft.X, pos.TopLeft.Y, c.wantX, c.wantY)
		}
	}
}

func TestGridPlacement_Place_NonSquareCount(t *testing.T) {
	p := grid.NewGridPlacement(100, 100, 10)

	// count=5 -> cols=ceil(sqrt(5))=3
	cases := []struct {
		index int
		wantX uint32
		wantY uint32
	}{
		{0, 11, 11}, // row 0, col 0
		{4, 44, 44}, // row 1, col 1
	}
	for _, c := range cases {
		pos := p.Place(c.index, 5)
		if pos.TopLeft.X != c.wantX || pos.TopLeft.Y != c.wantY {
			t.Errorf("Place(%d, 5) = (%d,%d), want (%d,%d)", c.index, pos.TopLeft.X, pos.TopLeft.Y, c.wantX, c.wantY)
		}
	}
}

func TestGridPlacement_Place_SingleEntity_Centers(t *testing.T) {
	p := grid.NewGridPlacement(100, 100, 10)
	pos := p.Place(0, 1)
	if pos.TopLeft.X != 45 || pos.TopLeft.Y != 45 {
		t.Errorf("Place(0, 1) = (%d,%d), want (45,45) (single cell spanning the whole grid, entity centered)", pos.TopLeft.X, pos.TopLeft.Y)
	}
}

func TestGridPlacement_ComponentsAndInit_AreNoops(t *testing.T) {
	p := grid.NewGridPlacement(100, 100, 10)
	if p.Components() != nil {
		t.Errorf("Components() = %v, want nil", p.Components())
	}
	p.Init(nil, 0, 0, 0) // must not panic
}
