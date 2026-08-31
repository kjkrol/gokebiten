package board

import "github.com/kjkrol/gokg/geom"

// CellID identifies one cell of a Grid — encoding is topology-specific.
type CellID uint64

// Grid abstracts a board's topology (square, hex, ...) behind neighbor,
// coordinate, and distance queries.
type Grid interface {
	Neighbors(c CellID) []CellID
	Contains(c CellID) bool
	CellCenter(c CellID) geom.Vec[float64]
	CellAt(pos geom.Vec[float64]) (CellID, bool)
	// Distance must never overestimate the true cost — it's the pathfinding heuristic.
	Distance(a, b CellID) float64
}
