package board

// Terrain reports one cell's terrain kind, independent of the Grid's topology.
type Terrain interface {
	Kind(c CellID) CellKind
}

// CellKind identifies a terrain kind (e.g. grass, wall, road) and its
// movement properties — the game defines its own named values (comparable
// via == / switch, since Name makes each one distinct).
type CellKind struct {
	Name string
	// Cost is relative to 1 (baseline); Passable gates entry entirely.
	Cost     float64
	Passable bool
}

// TerrainMap is a Terrain backed by a plain, gob-encodable map — mutate it
// directly (Set/SetMany) to change terrain at runtime, e.g. to build a road.
type TerrainMap struct {
	Cells   map[CellID]CellKind
	Default CellKind
}

var _ Terrain = (*TerrainMap)(nil)

func NewTerrainMap() *TerrainMap {
	return &TerrainMap{Cells: make(map[CellID]CellKind)}
}

func (t *TerrainMap) Kind(c CellID) CellKind {
	if kind, ok := t.Cells[c]; ok {
		return kind
	}
	return t.Default
}

// Set assigns c's terrain kind, taking effect immediately.
func (t *TerrainMap) Set(c CellID, kind CellKind) { t.Cells[c] = kind }

// SetMany assigns kind to every cell in cells in one call, instead of looping Set per cell.
func (t *TerrainMap) SetMany(cells []CellID, kind CellKind) {
	for _, c := range cells {
		t.Cells[c] = kind
	}
}

// SetAll resets every cell's terrain kind to kind, discarding any prior Set/SetMany overrides.
func (t *TerrainMap) SetAll(kind CellKind) {
	clear(t.Cells)
	t.Default = kind
}
