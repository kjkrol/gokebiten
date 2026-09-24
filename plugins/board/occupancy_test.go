package board_test

import (
	"testing"

	"github.com/kjkrol/gram/plugins/board"
)

func TestSingleOccupancy_IsOnePerDomainPerCell(t *testing.T) {
	o := &board.SingleOccupancy{}
	const cell, walker, hawk, boat, witch = board.CellID(7), 1, 2, 3, 4

	o.Enter(cell, walker, board.Land)
	if o.CanEnter(cell, boat, board.Land) {
		t.Error("a second land unit may enter a cell a land unit holds")
	}
	if !o.CanEnter(cell, hawk, board.Air) {
		t.Error("a flyer may not enter a cell a land unit holds")
	}
	if !o.CanEnter(cell, walker, board.Land) {
		t.Error("the holder itself may not re-enter")
	}
	o.Enter(cell, hawk, board.Air)
	if o.CanEnter(cell, witch, board.Land|board.Water) {
		t.Error("a unit of Land and Water may enter beside a land unit")
	}
	if !o.CanEnter(cell, boat, board.Water) {
		t.Error("a boat may not enter a cell held in Land and Air only")
	}
	o.Leave(cell, walker)
	if !o.CanEnter(cell, boat, board.Land) {
		t.Error("the land layer is still held after the land unit left")
	}
	if o.CanEnter(cell, boat, board.Air) {
		t.Error("the air layer is free while the flyer is still there")
	}
}

func TestMultipleOccupancy_LetsTokensStack(t *testing.T) {
	o := &board.MultipleOccupancy{}
	const cell = board.CellID(3)
	o.Enter(cell, 1, board.Land)
	o.Enter(cell, 2, board.Land)
	if !o.CanEnter(cell, 3, board.Land) {
		t.Error("a third token may not join two on a square")
	}
	o.Leave(cell, 1)
	o.Leave(cell, 2)
	if !o.CanEnter(cell, 3, board.Air) {
		t.Error("an empty square refuses")
	}
}
