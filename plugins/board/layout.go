package board

// Layout is a board's initial terrain for Plugin.Seed: Default fills every
// cell, then each CellEntry overrides one.
type Layout struct {
	Default string
	Cells   []CellEntry
}

// CellEntry sets Cell to the CellKind named Kind.
type CellEntry struct {
	Kind string
	Cell CellID
}
