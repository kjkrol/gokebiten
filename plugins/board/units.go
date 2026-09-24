package board

import (
	"fmt"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/gram/plugins/world/kind/comp"
)

// Units makes a game's unit kinds over the board: from where a row says the unit stands it
// derives its Position and Cell, from the kind's domain its Mover and Layers.
type Units[P any] struct {
	brd  *Plugin
	size float64
	at   func(row P) geom.Vec
}

// NewUnits binds a game's rows to the board: size is the unit's square box, at reads its centre
// from a row — a game that thinks in cells hands over their CellCenter.
func NewUnits[P any](brd *Plugin, size float64, at func(row P) geom.Vec) *Units[P] {
	return &Units[P]{brd: brd, size: size, at: at}
}

// Define registers one kind of unit: its name, the domain it moves in, its steering profile and
// whatever else the game gives its entities. It is kind.Define with the board's part filled in and
// the world's roster checked; a unit standing off the board panics when spawned.
func (u *Units[P]) Define(name string, domain Domain, steering world.Steering, extra ...comp.Comp) kind.Of[P] {
	brd := u.brd.Res.Logic.Board
	half := u.size / 2
	own := []comp.Comp{
		comp.Load(func(row P) world.Position {
			c := u.at(row)
			return world.Position{AABB: plane.NewAABB(geom.NewVec(c.X-half, c.Y-half), u.size, u.size)}
		}),
		comp.Load(func(row P) Cell {
			c, ok := brd.CellAt(u.at(row))
			if !ok {
				panic(fmt.Sprintf("board: a %q stands off the board at %v", name, u.at(row)))
			}
			return Cell{ID: c}
		}),
		comp.Const(Mover{Domain: domain}),
		comp.Const(world.Layers(domain)),
		comp.Const(steering),
	}
	spec := u.brd.worldPlugin.Roster().Unit.Spec(append(own, extra...)...)
	return kind.Define[P](u.brd.worldPlugin.Kinds(), name, spec)
}
