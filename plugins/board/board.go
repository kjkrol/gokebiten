package board

import (
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/gram/plugins/world"
)

// Board is a Grid paired with its TerrainMap — the single integration
// point for reading grid topology and reading/writing terrain kinds. It is also the world's
// Ground: a raster of the terrain's Altitude, one slot per cell, rebuilt when the terrain changes;
// on a square grid the ground runs smoothly between cells, each corner at the mean altitude of the
// cells round it, so a hill has slopes and a unit on a slope stands at its height.
type Board struct {
	Grid
	*TerrainMap

	heights []float64 // one per cell
	corners []float64 // one per lattice point of a square grid, (W+1) x (H+1); nil for other grids
	built   uint64
}

var _ world.Ground = (*Board)(nil)

// GroundAt is the ground height under p, 0 off the board: read between the cell's corners on a
// square grid, the cell's altitude elsewhere.
func (b *Board) GroundAt(p geom.Vec) float64 {
	c, ok := b.CellAt(p)
	if !ok {
		return 0
	}
	hs, x, y, sloped := b.Corners(c)
	if !sloped {
		return b.Altitude(c)
	}
	w, h := b.CellBounds()
	u := min(max((p.X-float64(x)*w)/w, 0), 1)
	v := min(max((p.Y-float64(y)*h)/h, 0), 1)
	return (1-u)*(1-v)*hs[0] + u*(1-v)*hs[1] + (1-u)*v*hs[2] + u*v*hs[3]
}

// Altitude is c's ground level, the Altitude of its kind read from the raster.
func (b *Board) Altitude(c CellID) float64 {
	b.refresh()
	i, ok := b.Ordinal(c)
	if !ok {
		return 0
	}
	return b.heights[i]
}

// Corners is the ground height at c's four corners — top-left, top-right, bottom-left,
// bottom-right — with c's column and row; false on a grid whose cells are flat.
func (b *Board) Corners(c CellID) (hs [4]float64, x, y uint32, ok bool) {
	b.refresh()
	if b.corners == nil {
		return hs, 0, 0, false
	}
	x, y, ok = b.Coords(c)
	if !ok {
		return hs, 0, 0, false
	}
	stride := int(b.Grid.(*squareGrid).Width) + 1
	at := func(dx, dy uint32) float64 { return b.corners[int(y+dy)*stride+int(x+dx)] }
	return [4]float64{at(0, 0), at(1, 0), at(0, 1), at(1, 1)}, x, y, true
}

func (b *Board) refresh() {
	if b.heights == nil || b.built != b.Version() {
		b.raster()
	}
}

// raster rebuilds the height tables from the terrain as it stands: the cells, and on a square
// grid the corners, each the mean of the cells that meet there.
func (b *Board) raster() {
	if b.heights == nil {
		b.heights = make([]float64, b.CellCount())
	}
	b.EachCell(func(c CellID) {
		if i, ok := b.Ordinal(c); ok {
			b.heights[i] = b.Kind(c).Altitude
		}
	})
	if sq, ok := b.Grid.(*squareGrid); ok {
		w, h := int(sq.Width), int(sq.Height)
		if b.corners == nil {
			b.corners = make([]float64, (w+1)*(h+1))
		}
		for cy := 0; cy <= h; cy++ {
			for cx := 0; cx <= w; cx++ {
				sum, n := 0.0, 0
				for _, d := range [4][2]int{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}} {
					x, y := cx+d[0], cy+d[1]
					if x < 0 || y < 0 || x >= w || y >= h {
						continue
					}
					sum, n = sum+b.heights[y*w+x], n+1
				}
				b.corners[cy*(w+1)+cx] = sum / float64(n)
			}
		}
	}
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
