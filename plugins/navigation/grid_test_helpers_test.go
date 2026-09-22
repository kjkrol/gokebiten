package navigation

import "github.com/kjkrol/gram/plugins/board"

func cellAtXY(g board.Grid, x, y uint32) board.CellID {
	c, _ := g.CellIndex(x, y)
	return c
}
