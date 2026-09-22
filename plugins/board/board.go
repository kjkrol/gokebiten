package board

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/gram/plugins/world"
)

// Board is a Grid paired with its TerrainMap — the single integration
// point for reading grid topology and reading/writing terrain kinds.
type Board struct {
	Grid
	*TerrainMap
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
