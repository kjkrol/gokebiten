package board

import "github.com/kjkrol/gram/render"

// Terrain reports one cell's terrain kind, independent of the Grid's topology.
type Terrain interface {
	Kind(c CellID) CellKind
}

// CellKind is a named terrain kind: its movement properties and the sprite drawn for it.
type CellKind struct {
	Name string
	// Cost 1 is full speed and the baseline path weight; above 1 the cell slows an entity and costs
	// more to plan through, below 1 is a boost — a game's choice, still capped by the move's step.
	Cost     float64
	Passable bool
	SpriteID render.SpriteID
}

// CellKindDict is a Plugin's registered set of CellKinds, keyed by Name —
// reached via Plugin.CellKindDict, never built directly by the game.
type CellKindDict interface {
	// Create registers kinds, assigning each one's SpriteID by call order.
	Create(kinds ...CellKind)
	// Get resolves name to the CellKind registered under it.
	Get(name string) (CellKind, bool)
	// All returns every registered CellKind.
	All() []CellKind
}

type cellKindDict struct {
	entries map[string]CellKind
	next    render.SpriteID
}

func newCellKindDict() *cellKindDict { return &cellKindDict{entries: make(map[string]CellKind)} }

func (d *cellKindDict) Create(kinds ...CellKind) {
	for _, k := range kinds {
		k.SpriteID = d.next
		d.next++
		d.entries[k.Name] = k
	}
}

func (d *cellKindDict) Get(name string) (CellKind, bool) {
	k, ok := d.entries[name]
	return k, ok
}

func (d *cellKindDict) All() []CellKind {
	all := make([]CellKind, 0, len(d.entries))
	for _, k := range d.entries {
		all = append(all, k)
	}
	return all
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
