package board

import "github.com/kjkrol/aabbworld/geom"

// CellID identifies one cell of a Grid — encoding is topology-specific.
type CellID uint64

// Grid abstracts a board's topology (square, hex, ...) behind neighbor,
// coordinate, and distance queries.
type Grid interface {
	Neighbors(c CellID) []CellID
	Contains(c CellID) bool
	CellCenter(c CellID) geom.Vec
	CellAt(pos geom.Vec) (CellID, bool)
	// CellIndex returns the CellID at grid coordinates (col,row or axial q,r), if within bounds.
	CellIndex(a, b uint32) (CellID, bool)
	// NeighborCost is the geometric step cost from a to its neighbor b.
	NeighborCost(a, b CellID) float64
	// DiagonalNeighbors returns the two cells flanking the corner between a and its diagonal b.
	DiagonalNeighbors(a, b CellID) (c1, c2 CellID, ok bool)
	// Distance must never overestimate the true cost — it's the pathfinding heuristic.
	Distance(a, b CellID) float64
	// CellSpan is the world-space side length of one cell — the renderer's cell-quad size.
	CellSpan() float32
}

type wrapSetter interface {
	SetWrap(x, y bool)
}

// foldAxis maps v onto [0,size): wrapped when the axis wraps, refused outside it otherwise.
func foldAxis(v, size int64, wraps bool) (int64, bool) {
	if wraps {
		return wrapModI64(v, size), true
	}
	return v, v >= 0 && v < size
}
