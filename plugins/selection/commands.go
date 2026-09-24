package selection

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/gram/plugins/players"
	"github.com/kjkrol/uid"
)

// Select is the command to select: exactly IDs when given, else every Selectable entity in Box
// (world units); Additive keeps what was selected before.
type Select struct {
	IDs      []uid.UID64
	Box      geom.AABB
	Additive bool
}

// DefaultBindings is left drag (a click is a drag of no length) into a Select of the box it drew,
// Shift for an additive one. Bind them on a player.
func DefaultBindings() []players.Binding {
	box := func(additive bool) func(c players.Context) (Select, bool) {
		return func(c players.Context) (Select, bool) {
			return Select{Box: c.WorldBox(c.Start, c.Cursor), Additive: additive}, true
		}
	}
	return []players.Binding{
		players.Command(players.Drag{Button: ebiten.MouseButtonLeft}, "Select", box(false)),
		players.Command(players.Drag{Button: ebiten.MouseButtonLeft, Mods: players.Mods{Shift: true}}, "Add to selection", box(true)),
	}
}
