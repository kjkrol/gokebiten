package board

type DefaultGrids struct{}

func (DefaultGrids) Square(width, height, cellSize uint32) Grid {
	return newSquareGrid(width, height, cellSize)
}

func (DefaultGrids) Hex(width, height uint32, size float64) Grid {
	return newHexGrid(width, height, size)
}
