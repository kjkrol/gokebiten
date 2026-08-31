package board

// Terrain reports the movement cost and passability of a cell, independent
// of the Grid's topology.
type Terrain interface {
	// MovementCost's cost is relative to 1 (baseline); passable gates entry entirely.
	MovementCost(c CellID) (cost float64, passable bool)
}

// CellProps is one cell's terrain data in a TerrainMap.
type CellProps struct {
	Cost     float64
	Passable bool
}

// TerrainMap is a Terrain backed by a plain, gob-encodable map — mutate it
// directly (Set) to change terrain at runtime, e.g. to build a road.
type TerrainMap struct {
	Cells   map[CellID]CellProps
	Default CellProps
}

var _ Terrain = (*TerrainMap)(nil)

func NewTerrainMap(defaultProps CellProps) *TerrainMap {
	return &TerrainMap{Cells: make(map[CellID]CellProps), Default: defaultProps}
}

func (t *TerrainMap) MovementCost(c CellID) (float64, bool) {
	if props, ok := t.Cells[c]; ok {
		return props.Cost, props.Passable
	}
	return t.Default.Cost, t.Default.Passable
}

// Set assigns c's terrain properties, taking effect immediately.
func (t *TerrainMap) Set(c CellID, props CellProps) { t.Cells[c] = props }
