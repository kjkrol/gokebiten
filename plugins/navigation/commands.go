package navigation

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/players"
)

// MoveTo is the command to send every Selected entity to Cell, or with Append to add Cell behind
// the orders they already have.
type MoveTo struct {
	Cell   board.CellID
	Append bool
}

// DefaultBindings is a right click into a MoveTo of the cell under the cursor, Shift to append.
// Bind them on a player.
func (p *Plugin) DefaultBindings() []players.Binding {
	grid := p.boardPlugin.Res.Logic.Board.Grid
	to := func(appendIt bool) func(c players.Context) (MoveTo, bool) {
		return func(c players.Context) (MoveTo, bool) {
			cell, ok := grid.CellAt(c.World(c.Cursor))
			return MoveTo{Cell: cell, Append: appendIt}, ok
		}
	}
	return []players.Binding{
		players.Command(players.ButtonPress{Button: ebiten.MouseButtonRight}, "Move selected units here", to(false)),
		players.Command(players.ButtonPress{Button: ebiten.MouseButtonRight, Mods: players.Mods{Shift: true}}, "Add a waypoint", to(true)),
	}
}
