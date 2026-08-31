package board

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// TerrainSpeedModifier scales Velocity by 1/cost for whichever cell an
// entity currently occupies — implements world.SpeedModifier.
type TerrainSpeedModifier struct {
	grid    Grid
	terrain Terrain
	pos     goke.OptComp[world.Position]
}

var _ world.SpeedModifier = (*TerrainSpeedModifier)(nil)

func NewTerrainSpeedModifier(grid Grid, terrain Terrain) *TerrainSpeedModifier {
	return &TerrainSpeedModifier{grid: grid, terrain: terrain}
}

func (t *TerrainSpeedModifier) Bind(qb *goke.QueryBuilder) { qb.Optional(&t.pos) }

func (t *TerrainSpeedModifier) Apply(cur *goke.Cursor, i int, acc float64) float64 {
	positions := t.pos.Slice(cur)
	if positions == nil {
		return acc
	}
	cell, ok := t.grid.CellAt(center(positions[i]))
	if !ok {
		return acc
	}
	cost, passable := t.terrain.MovementCost(cell)
	if !passable || cost <= 0 {
		return acc
	}
	return acc / cost
}
