package board

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/world"
)

// TerrainSpeedModifier scales the speed by 1/Cost of the cell under the entity's centre.
type TerrainSpeedModifier struct {
	grid    Grid
	terrain Terrain
}

var _ world.SpeedModifier = (*TerrainSpeedModifier)(nil)

func NewTerrainSpeedModifier(grid Grid, terrain Terrain) *TerrainSpeedModifier {
	return &TerrainSpeedModifier{grid: grid, terrain: terrain}
}

func (t *TerrainSpeedModifier) Bind(*goke.QueryBuilder) {}

func (t *TerrainSpeedModifier) Apply(_ *goke.Cursor, _ int, base *world.Base, acc float64) float64 {
	cell, ok := t.grid.CellAt(Center(base.Pos))
	if !ok {
		return acc
	}
	kind := t.terrain.Kind(cell)
	if kind.Cost <= 0 {
		return acc
	}
	return acc / kind.Cost
}
