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
	// CellIndex returns the CellID at the topology-specific coordinate pair
	// (col,row for a square grid; axial q,r for a hex grid) — the direct,
	// position-free counterpart to CellAt. ok is false only when out of
	// bounds on a non-toroidal grid (a toroidal grid always wraps).
	CellIndex(a, b uint32) (CellID, bool)
	// NeighborCost is the geometric step cost from a to its neighbor b (as
	// returned by Neighbors) — 1 where every neighbor is equidistant (hex);
	// varies for a square grid with diagonal movement enabled. Undefined for
	// a non-adjacent pair.
	NeighborCost(a, b CellID) float64
	// DiagonalNeighbors returns the two orthogonal cells flanking the corner
	// between a and its diagonal neighbor b — ok is false if b isn't a
	// diagonal neighbor of a (e.g. any pair on a hex grid, which has no
	// diagonal/corner concept). Used to block cutting through a corner where
	// either flanking cell is impassable.
	DiagonalNeighbors(a, b CellID) (c1, c2 CellID, ok bool)
	// Distance must never overestimate the true cost — it's the pathfinding heuristic.
	Distance(a, b CellID) float64
	// CellSpan is the world-space side length of one cell — the renderer's cell-quad size.
	CellSpan() float32
}
