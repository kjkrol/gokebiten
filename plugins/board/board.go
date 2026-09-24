package board

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/gram/plugins/world"
)

// Board is a Grid paired with its TerrainMap — the single integration
// point for reading grid topology and reading/writing terrain kinds. It is also the world's
// Ground: a raster of the terrain's Altitude, one slot per cell, rebuilt when the terrain changes.
type Board struct {
	Grid
	*TerrainMap

	heights []float64
	built   uint64
}

var _ world.Ground = (*Board)(nil)

// GroundAt is the ground height under p, 0 off the board.
func (b *Board) GroundAt(p geom.Vec) float64 {
	c, ok := b.CellAt(p)
	if !ok {
		return 0
	}
	return b.Altitude(c)
}

// Altitude is c's ground height, the Altitude of its kind read from the raster.
func (b *Board) Altitude(c CellID) float64 {
	if b.heights == nil || b.built != b.Version() {
		b.raster()
	}
	i, ok := b.Ordinal(c)
	if !ok {
		return 0
	}
	return b.heights[i]
}

// raster rebuilds the height table from the terrain as it stands.
func (b *Board) raster() {
	if b.heights == nil {
		b.heights = make([]float64, b.CellCount())
	}
	b.EachCell(func(c CellID) {
		if i, ok := b.Ordinal(c); ok {
			b.heights[i] = b.Kind(c).Altitude
		}
	})
	b.built = b.Version()
}

// At is GroundAt — the world.Ground contract.
func (b *Board) At(p geom.Vec) float64 { return b.GroundAt(p) }

// Step is how far apart sight samples the ground: the shorter side of a cell.
func (b *Board) Step() float64 {
	w, h := b.CellBounds()
	return min(w, h)
}

// Cell is an entity's current position on the board.
type Cell struct{ ID CellID }

func NewBoard(grid Grid, terrain *TerrainMap) *Board {
	return &Board{Grid: grid, TerrainMap: terrain}
}

// CellAABB is the size x size world rectangle centred on c.
func CellAABB(grid Grid, c CellID, size uint32) plane.AABB {
	center := grid.CellCenter(c)
	half := float64(size) / 2
	topLeft := geom.NewVec(center.X-half, center.Y-half)
	return plane.NewAABB(topLeft, float64(size), float64(size))
}

// Center returns pos's world-space center point.
func Center(pos world.Position) geom.Vec {
	return geom.NewVec(float64(pos.TopLeft.X)+float64(pos.Size.X)/2, float64(pos.TopLeft.Y)+float64(pos.Size.Y)/2)
}
