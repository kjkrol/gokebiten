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
	// CellBounds is the width and height of the rectangle round one cell — the drawn quad.
	CellBounds() (w, h float64)
	// CellOutline appends to dst the corners of c, in order round the cell.
	CellOutline(c CellID, dst []geom.Vec) []geom.Vec
	// CellBoxes appends to dst boxes that together cover c, exactly or from outside.
	CellBoxes(c CellID, dst []geom.AABB) []geom.AABB
	// CellsUnder calls fn for every cell the box touches, each once.
	CellsUnder(box geom.AABB, fn func(c CellID))
	// EachCell calls fn for every cell of the grid.
	EachCell(fn func(c CellID))
}

type wrapSetter interface {
	SetWrap(x, y bool)
}

// cellsUnder is CellsUnder for any grid: candidates from CellAt on a lattice over the box grown by
// half a cell, kept when their outline meets the box.
func cellsUnder(g Grid, box geom.AABB, fn func(c CellID)) {
	w, h := g.CellBounds()
	step := min(w, h) / 2
	if step <= 0 {
		return
	}
	seen := map[CellID]struct{}{}
	var outline []geom.Vec
	for y := box.TopLeft.Y - h/2; y <= box.BottomRight.Y+h/2; y += step {
		for x := box.TopLeft.X - w/2; x <= box.BottomRight.X+w/2; x += step {
			c, ok := g.CellAt(geom.NewVec(x, y))
			if !ok {
				continue
			}
			if _, dup := seen[c]; dup {
				continue
			}
			seen[c] = struct{}{}
			outline = g.CellOutline(c, outline[:0])
			if polygonMeetsBox(outline, box) {
				fn(c)
			}
		}
	}
}

// polygonMeetsBox reports whether a convex polygon and a box share any point.
func polygonMeetsBox(poly []geom.Vec, box geom.AABB) bool {
	const eps = 1e-9
	for _, p := range poly {
		if p.X >= box.TopLeft.X-eps && p.X <= box.BottomRight.X+eps && p.Y >= box.TopLeft.Y-eps && p.Y <= box.BottomRight.Y+eps {
			return true
		}
	}
	corners := [4]geom.Vec{box.TopLeft, geom.NewVec(box.BottomRight.X, box.TopLeft.Y), box.BottomRight, geom.NewVec(box.TopLeft.X, box.BottomRight.Y)}
	for _, c := range corners {
		if pointInConvex(poly, c) {
			return true
		}
	}
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		for j := range corners {
			if segmentsCross(a, b, corners[j], corners[(j+1)%4]) {
				return true
			}
		}
	}
	return false
}

// pointInConvex reports whether p lies in the convex polygon poly, given in one winding order.
func pointInConvex(poly []geom.Vec, p geom.Vec) bool {
	sign := 0.0
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		cross := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X)
		switch {
		case cross == 0:
		case sign == 0:
			sign = cross
		case cross*sign < 0:
			return false
		}
	}
	return true
}

// segmentsCross reports whether segments ab and cd intersect.
func segmentsCross(a, b, c, d geom.Vec) bool {
	orient := func(p, q, r geom.Vec) float64 { return (q.X-p.X)*(r.Y-p.Y) - (q.Y-p.Y)*(r.X-p.X) }
	o1, o2 := orient(a, b, c), orient(a, b, d)
	o3, o4 := orient(c, d, a), orient(c, d, b)
	return o1*o2 < 0 && o3*o4 < 0
}

// foldAxis maps v onto [0,size): wrapped when the axis wraps, refused outside it otherwise.
func foldAxis(v, size int64, wraps bool) (int64, bool) {
	if wraps {
		return wrapModI64(v, size), true
	}
	return v, v >= 0 && v < size
}
