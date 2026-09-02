package board

import (
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// Board is a Grid paired with its Terrain — no dynamic entity state, which
// lives in whichever Occupancy this plugin was built with.
type Board struct {
	Grid
	Terrain
}

// Cell is an entity's current position on the board.
type Cell struct{ ID CellID }

func NewBoard(grid Grid, terrain Terrain) *Board {
	return &Board{Grid: grid, Terrain: terrain}
}

// CellAABB is the size x size world-pixel rectangle centered on c, the same placement plugins/navigation's NavigationSystem steps entities into.
func CellAABB(grid Grid, c CellID, size uint32) plane.AABB[uint32] {
	center := grid.CellCenter(c)
	half := float64(size) / 2
	topLeft := geom.NewVec(uint32(center.X-half), uint32(center.Y-half))
	return plane.NewAABB(topLeft, size, size)
}

// Center returns pos's world-space center point.
func Center(pos world.Position) geom.Vec[float64] {
	return geom.NewVec(float64(pos.TopLeft.X)+float64(pos.Size.X)/2, float64(pos.TopLeft.Y)+float64(pos.Size.Y)/2)
}
