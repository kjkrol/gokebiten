package navigation

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/board"
)

// MoveTo is the command to send every Selected entity to Cell, or with Append to add Cell behind
// the orders they already have.
type MoveTo struct {
	Cell   board.CellID
	Append bool
}

var _ plugin.Commander = (*Plugin)(nil)

// Commands is the inbox MoveTo lands in — for the players plugin.
func (p *Plugin) Commands() []control.Mailbox { return []control.Mailbox{&p.moves} }

// DefaultBindings is a right click into a MoveTo of the cell under the cursor, Shift to append.
func (p *Plugin) DefaultBindings() []control.Binding {
	grid := p.boardPlugin.Res.Logic.Board.Grid
	to := func(appendIt bool) func(c control.Context) (MoveTo, bool) {
		return func(c control.Context) (MoveTo, bool) {
			cell, ok := grid.CellAt(c.World(c.Cursor))
			return MoveTo{Cell: cell, Append: appendIt}, ok
		}
	}
	return []control.Binding{
		control.Command(control.ButtonPress{Button: ebiten.MouseButtonRight}, "Move selected units here", to(false)),
		control.Command(control.ButtonPress{Button: ebiten.MouseButtonRight, Mods: control.Mods{Shift: true}}, "Add a waypoint", to(true)),
	}
}
