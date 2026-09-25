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
// derives its Position and Cell, from the kind's Mover its Layers, and in a Quasi3D world its Z
// from the Shape.
type Units[P any] struct {
	brd   *Plugin
	shape Shape
	at    func(row P) geom.Vec
}

// Shape is a unit's body: the side of its square box and, in a Quasi3D world, how tall it stands.
type Shape struct{ Size, Height float64 }

// NewUnits binds a game's rows to the board: shape is the units' body, at reads a unit's centre
// from its row — a game that thinks in cells hands over their CellCenter.
func NewUnits[P any](brd *Plugin, shape Shape, at func(row P) geom.Vec) *Units[P] {
	if !brd.worldPlugin.Quasi3D() && shape.Height != 0 {
		panic("board: units with a Height in a flat world; set world.Config.Quasi3D")
	}
	return &Units[P]{brd: brd, shape: shape, at: at}
}

// Define registers one kind of unit: its name, how it moves (the domains, and in a Quasi3D world
// the Lift it keeps above the ground), its steering profile and whatever else the game gives its
// entities. It is kind.Define with the board's part filled in and the world's roster checked; a
// unit standing off the board panics when spawned.
func (u *Units[P]) Define(name string, mover Mover, steering world.Steering, extra ...comp.Comp) kind.Of[P] {
	brd := u.brd.Res.Logic.Board
	quasi3D := u.brd.worldPlugin.Quasi3D()
	if !quasi3D && mover.Lift != 0 {
		panic(fmt.Sprintf("board: %q has a Lift in a flat world; set world.Config.Quasi3D", name))
	}
	half := u.shape.Size / 2
	own := []comp.Comp{
		comp.Load(func(row P) world.Position {
			c := u.at(row)
			return world.Position{AABB: plane.NewAABB(geom.NewVec(c.X-half, c.Y-half), u.shape.Size, u.shape.Size)}
		}),
		comp.Load(func(row P) Cell {
			c, ok := brd.CellAt(u.at(row))
			if !ok {
				panic(fmt.Sprintf("board: a %q stands off the board at %v", name, u.at(row)))
			}
			return Cell{ID: c}
		}),
		comp.Const(mover),
		comp.Const(world.Layers(mover.Domain)),
		comp.Const(steering),
	}
	if quasi3D {
		own = append(own, comp.Const(world.Z{Height: u.shape.Height})) // Altitude is the board's to write
	}
	spec := u.brd.worldPlugin.Roster().Unit.Spec(append(own, extra...)...)
	return kind.Define[P](u.brd.worldPlugin.Kinds(), name, spec)
}
