package navigation

import "github.com/kjkrol/gokebiten/plugins/board"

// Resources is navigation's single published Resources — live move-order
// state HandleEvents writes to and CommandSystem reads/clears.
type Resources struct {
	PendingTarget *board.CellID
}
