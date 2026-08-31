package board

import (
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// Board is a Grid paired with its Terrain — no dynamic entity state, which
// lives in whichever Occupancy the Movement module was built with.
type Board struct {
	Grid
	Terrain
}

func NewBoard(grid Grid, terrain Terrain) *Board {
	return &Board{Grid: grid, Terrain: terrain}
}

// CellAABB is the size x size world-pixel rectangle centered on c, the same placement Movement steps entities into.
func CellAABB(grid Grid, c CellID, size uint32) plane.AABB[uint32] {
	center := grid.CellCenter(c)
	half := float64(size) / 2
	topLeft := geom.NewVec(uint32(center.X-half), uint32(center.Y-half))
	return plane.NewAABB(topLeft, size, size)
}
