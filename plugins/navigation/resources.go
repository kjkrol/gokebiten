package navigation

import "github.com/kjkrol/gram/plugins/board"

// Resources is navigation's single published Resources — live move-order
// state HandleEvents writes to and CommandSystem reads/clears.
type Resources struct {
	Pending *MoveCommand
}

// MoveCommand is a right-click's target and whether it is added after the current orders.
type MoveCommand struct {
	Cell   board.CellID
	Append bool
}
