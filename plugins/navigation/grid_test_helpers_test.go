package navigation

import "github.com/kjkrol/gokebiten/plugins/board"

func cellAtXY(g *board.SquareGrid, x, y uint32) board.CellID {
	c, _ := g.CellIndex(x, y)
	return c
}
