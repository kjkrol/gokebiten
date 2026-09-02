package board

import (
	"math"

	"github.com/kjkrol/gokg/geom"
)

// HexGrid is a Grid over pointy-top hexagonal cells addressed by axial
// coordinates, bounded to a Width x Height parallelogram.
type HexGrid struct {
	Width, Height uint32
	Size          float64
	Toroidal      bool
}

var _ Grid = (*HexGrid)(nil)

func NewHexGrid(width, height uint32, size float64) *HexGrid {
	return &HexGrid{Width: width, Height: height, Size: size}
}

func packAxial(q, r int32) CellID {
	return CellID(uint64(uint32(q))<<32 | uint64(uint32(r)))
}

func unpackAxial(c CellID) (q, r int32) {
	return int32(uint32(c >> 32)), int32(uint32(c))
}

// wrapAxial folds q,r into the canonical [0,Width) x [0,Height) parallelogram.
func (g *HexGrid) wrapAxial(q, r int32) (int32, int32) {
	return int32(wrapModI64(int64(q), int64(g.Width))), int32(wrapModI64(int64(r), int64(g.Height)))
}

// Contains is always true when Toroidal — every (q,r) maps to some cell once wrapped.
func (g *HexGrid) Contains(c CellID) bool {
	if g.Toroidal {
		return true
	}
	q, r := unpackAxial(c)
	return q >= 0 && r >= 0 && uint32(q) < g.Width && uint32(r) < g.Height
}

var hexDirs = [6][2]int32{{1, 0}, {1, -1}, {0, -1}, {-1, 0}, {-1, 1}, {0, 1}}

func (g *HexGrid) Neighbors(c CellID) []CellID {
	q, r := unpackAxial(c)
	out := make([]CellID, 0, 6)
	for _, d := range hexDirs {
		nq, nr := q+d[0], r+d[1]
		if g.Toroidal {
			wq, wr := g.wrapAxial(nq, nr)
			out = append(out, packAxial(wq, wr))
			continue
		}
		if n := packAxial(nq, nr); g.Contains(n) {
			out = append(out, n)
		}
	}
	return out
}

// CellCenter shifts the standard axial-to-pixel conversion so cell (0,0) sits fully in positive space.
func (g *HexGrid) CellCenter(c CellID) geom.Vec[float64] {
	q, r := unpackAxial(c)
	x := g.Size*(math.Sqrt(3)*float64(q)+math.Sqrt(3)/2*float64(r)) + g.Size
	y := g.Size*(1.5*float64(r)) + g.Size
	return geom.NewVec(x, y)
}

func (g *HexGrid) CellAt(pos geom.Vec[float64]) (CellID, bool) {
	if g.Size == 0 {
		return 0, false
	}
	x, y := pos.X-g.Size, pos.Y-g.Size
	qf := (math.Sqrt(3)/3*x - 1.0/3*y) / g.Size
	rf := (2.0 / 3 * y) / g.Size
	q, r := axialRound(qf, rf)
	if g.Toroidal {
		wq, wr := g.wrapAxial(q, r)
		return packAxial(wq, wr), true
	}
	c := packAxial(q, r)
	if !g.Contains(c) {
		return 0, false
	}
	return c, true
}

func (g *HexGrid) CellIndex(q, r uint32) (CellID, bool) {
	if g.Toroidal {
		return packAxial(int32(q%g.Width), int32(r%g.Height)), true
	}
	if q >= g.Width || r >= g.Height {
		return 0, false
	}
	return packAxial(int32(q), int32(r)), true
}

// NeighborCost is always 1 — every hex neighbor is equidistant in this axial model.
func (g *HexGrid) NeighborCost(a, b CellID) float64 { return 1 }

// DiagonalNeighbors always returns ok=false — hex neighbors share a full
// edge, not a corner point, so cutting through a corner isn't a concept here.
func (g *HexGrid) DiagonalNeighbors(a, b CellID) (c1, c2 CellID, ok bool) {
	return 0, 0, false
}

func hexCubeDistance(dq, dr int32) float64 {
	return (math.Abs(float64(dq)) + math.Abs(float64(dr)) + math.Abs(float64(dq+dr))) / 2
}

// Distance is hex (cube) distance; when Toroidal it checks every wrap period and returns the shortest.
func (g *HexGrid) Distance(a, b CellID) float64 {
	aq, ar := unpackAxial(a)
	bq, br := unpackAxial(b)
	dq, dr := aq-bq, ar-br

	if !g.Toroidal {
		return hexCubeDistance(dq, dr)
	}
	width, height := int32(g.Width), int32(g.Height)
	best := math.Inf(1)
	for _, mq := range [3]int32{-width, 0, width} {
		for _, mr := range [3]int32{-height, 0, height} {
			if d := hexCubeDistance(dq+mq, dr+mr); d < best {
				best = d
			}
		}
	}
	return best
}

// axialRound snaps fractional cube coordinates to the nearest valid hex.
func axialRound(qf, rf float64) (int32, int32) {
	xf, zf := qf, rf
	yf := -xf - zf
	x, y, z := math.Round(xf), math.Round(yf), math.Round(zf)
	dx, dy, dz := math.Abs(x-xf), math.Abs(y-yf), math.Abs(z-zf)
	switch {
	case dx > dy && dx > dz:
		x = -y - z
	case dy > dz:
	default:
		z = -x - y
	}
	return int32(x), int32(z)
}
