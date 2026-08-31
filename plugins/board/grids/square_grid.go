package grids

import (
	"math"

	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokg/geom"
)

// SquareGrid is a board.Grid over a Width x Height array of square cells,
// CellSize world-pixels on a side, with 4-directional (N/S/E/W) neighbors.
type SquareGrid struct {
	Width, Height uint32
	CellSize      uint32
	Toroidal      bool
}

var _ board.Grid = (*SquareGrid)(nil)

func NewSquareGrid(width, height, cellSize uint32) *SquareGrid {
	return &SquareGrid{Width: width, Height: height, CellSize: cellSize}
}

func (g *SquareGrid) idAt(x, y uint32) board.CellID {
	return board.CellID(uint64(y)*uint64(g.Width) + uint64(x))
}

func (g *SquareGrid) cellXY(c board.CellID) (x, y uint32) {
	return uint32(uint64(c) % uint64(g.Width)), uint32(uint64(c) / uint64(g.Width))
}

func (g *SquareGrid) Contains(c board.CellID) bool {
	if g.Toroidal {
		return true
	}
	x, y := g.cellXY(c)
	return uint64(c) < uint64(g.Width)*uint64(g.Height) && x < g.Width && y < g.Height
}

var squareDirs = [4][2]int64{{0, -1}, {0, 1}, {-1, 0}, {1, 0}}

func (g *SquareGrid) Neighbors(c board.CellID) []board.CellID {
	x, y := g.cellXY(c)
	out := make([]board.CellID, 0, 4)
	for _, d := range squareDirs {
		nx, ny := int64(x)+d[0], int64(y)+d[1]
		if g.Toroidal {
			nx, ny = wrapModI64(nx, int64(g.Width)), wrapModI64(ny, int64(g.Height))
		} else if nx < 0 || ny < 0 || nx >= int64(g.Width) || ny >= int64(g.Height) {
			continue
		}
		out = append(out, g.idAt(uint32(nx), uint32(ny)))
	}
	return out
}

func (g *SquareGrid) CellCenter(c board.CellID) geom.Vec[float64] {
	x, y := g.cellXY(c)
	half := float64(g.CellSize) / 2
	return geom.NewVec(float64(x)*float64(g.CellSize)+half, float64(y)*float64(g.CellSize)+half)
}

func (g *SquareGrid) CellAt(pos geom.Vec[float64]) (board.CellID, bool) {
	if g.CellSize == 0 {
		return 0, false
	}
	if !g.Toroidal && (pos.X < 0 || pos.Y < 0) {
		return 0, false
	}
	x := int64(math.Floor(pos.X / float64(g.CellSize)))
	y := int64(math.Floor(pos.Y / float64(g.CellSize)))
	if g.Toroidal {
		return g.idAt(uint32(wrapModI64(x, int64(g.Width))), uint32(wrapModI64(y, int64(g.Height)))), true
	}
	if x < 0 || y < 0 || uint32(x) >= g.Width || uint32(y) >= g.Height {
		return 0, false
	}
	return g.idAt(uint32(x), uint32(y)), true
}

// Distance is Manhattan distance, admissible for this grid's 4-directional Neighbors.
func (g *SquareGrid) Distance(a, b board.CellID) float64 {
	ax, ay := g.cellXY(a)
	bx, by := g.cellXY(b)
	width, height := uint32(0), uint32(0)
	if g.Toroidal {
		width, height = g.Width, g.Height
	}
	return float64(wrapDistance1D(ax, bx, width) + wrapDistance1D(ay, by, height))
}

func absDiffU32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

// wrapDistance1D is the shorter of the direct gap between a and b on an axis
// of length size, or the gap going the other way around it (size 0 disables wrapping).
func wrapDistance1D(a, b, size uint32) uint32 {
	d := absDiffU32(a, b)
	if size == 0 {
		return d
	}
	if alt := size - d; alt < d {
		return alt
	}
	return d
}

// wrapModI64 folds v into [0,m) using true (non-negative) modulo.
func wrapModI64(v, m int64) int64 {
	v %= m
	if v < 0 {
		v += m
	}
	return v
}
