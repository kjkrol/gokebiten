package grids

import "testing"

func TestHexGrid_NonToroidal_EdgeExcludesNeighbors(t *testing.T) {
	g := NewHexGrid(4, 4, 10)
	corner := g.Neighbors(packAxial(0, 0))
	if len(corner) == 6 {
		t.Error("corner cell has 6 neighbors, want fewer (no wrap)")
	}
}

func TestHexGrid_Toroidal_AlwaysSixNeighbors(t *testing.T) {
	g := &HexGrid{Width: 4, Height: 4, Size: 10, Toroidal: true}
	for q := int32(0); q < int32(g.Width); q++ {
		for r := int32(0); r < int32(g.Height); r++ {
			n := g.Neighbors(packAxial(q, r))
			if len(n) != 6 {
				t.Errorf("cell (%d,%d) has %d neighbors, want 6", q, r, len(n))
			}
		}
	}
}

func TestHexGrid_Toroidal_DistanceWrapsAtEdge(t *testing.T) {
	g := &HexGrid{Width: 4, Height: 4, Size: 10, Toroidal: true}
	first := packAxial(0, 0)
	last := packAxial(int32(g.Width-1), 0)

	if d := g.Distance(first, last); d != 1 {
		t.Errorf("Distance(q=0, q=Width-1) = %v, want 1 (wrap-around, not %d)", d, g.Width-1)
	}
}

func TestHexGrid_Toroidal_ContainsAlwaysTrue(t *testing.T) {
	g := &HexGrid{Width: 4, Height: 4, Size: 10, Toroidal: true}
	if !g.Contains(packAxial(999, -999)) {
		t.Error("expected Contains to always be true on a toroidal grid")
	}
}

func TestHexGrid_Toroidal_NeighborsWrapToCanonicalRange(t *testing.T) {
	g := &HexGrid{Width: 4, Height: 4, Size: 10, Toroidal: true}
	for _, n := range g.Neighbors(packAxial(0, 0)) {
		q, r := unpackAxial(n)
		if q < 0 || r < 0 || uint32(q) >= g.Width || uint32(r) >= g.Height {
			t.Errorf("neighbor (%d,%d) is outside the canonical range, want it wrapped", q, r)
		}
	}
}
