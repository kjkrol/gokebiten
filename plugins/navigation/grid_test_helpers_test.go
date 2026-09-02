package navigation_test

import (
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/board/grids"
)

func cellAtXY(g *grids.SquareGrid, x, y uint32) board.CellID {
	c, _ := g.CellIndex(x, y)
	return c
}
