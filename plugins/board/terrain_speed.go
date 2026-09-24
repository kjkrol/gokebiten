package board

import (
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/world"
)

// terrainSpeed is the Moving behavior board registers on the world: every entity carrying a Mover
// moves at 1/CostFor(its domain) of the cell under its centre.
func terrainSpeed(grid Grid, terrain Terrain) plugin.Behavior {
	return plugin.Each[Mover](func(_ plugin.Tick, m *Mover, mv world.Moving) {
		cell, ok := grid.CellAt(Center(mv.Base.Pos))
		if !ok {
			return
		}
		if cost := terrain.Kind(cell).CostFor(m.Domain); cost > 0 {
			mv.Base.Vel.Value /= cost
		}
	})
}
