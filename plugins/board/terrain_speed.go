package board

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/world"
)

// TerrainSpeedModifier scales the speed by 1/CostFor(domain) of the cell under the entity's
// centre, the domain read from its Mover (Land without one).
type TerrainSpeedModifier struct {
	grid    Grid
	terrain Terrain
	mover   goke.OptComp[Mover]
}

var _ world.SpeedModifier = (*TerrainSpeedModifier)(nil)

func NewTerrainSpeedModifier(grid Grid, terrain Terrain) *TerrainSpeedModifier {
	return &TerrainSpeedModifier{grid: grid, terrain: terrain}
}

func (t *TerrainSpeedModifier) Bind(qb *goke.QueryBuilder) { qb.Optional(&t.mover) }

func (t *TerrainSpeedModifier) Apply(cur *goke.Cursor, i int, base *world.Base, acc float64) float64 {
	cell, ok := t.grid.CellAt(Center(base.Pos))
	if !ok {
		return acc
	}
	cost := t.terrain.Kind(cell).CostFor(DomainAt(t.mover.Slice(cur), i))
	if cost <= 0 {
		return acc
	}
	return acc / cost
}
