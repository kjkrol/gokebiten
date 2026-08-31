package board_test

import (
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/board/grids"
	"github.com/kjkrol/gokg/geom"
)

func cellAtXY(g *grids.SquareGrid, x, y uint32) board.CellID {
	c, _ := g.CellAt(geom.NewVec(float64(x)*float64(g.CellSize)+1, float64(y)*float64(g.CellSize)+1))
	return c
}
