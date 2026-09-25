package world

import "github.com/kjkrol/aabbworld/geom"

// Z is an entity's place in height: Altitude is its bottom, written by the board from the ground
// under it, Height its rise above that. Only a Quasi3D world carries it; see Config.Quasi3D.
type Z struct{ Altitude, Height float64 }

// Top is the entity's highest point.
func (z Z) Top() float64 { return z.Altitude + z.Height }

// Ground is the height of the world's ground at a point and how far apart it is sampled along a
// ray; the board gives a Quasi3D world one, sight reads it.
type Ground interface {
	At(p geom.Vec) float64
	Step() float64
}
