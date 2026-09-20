package collisions

import "math"

// Physics is what makes an entity take part in the physical side of a contact —
// pushed out of an overlap, and bouncing. One without it is only ever detected.
//
// The zero value weighs DefaultMass and does not bounce at all, so an entity
// meant to rebound has to say so: Physics{Restitution: 1}.
type Physics struct {
	// Mass is how hard the entity is to shift, in arbitrary units. A
	// non-positive Mass weighs DefaultMass; +Inf is never moved by anything,
	// and everything that hits it is pushed out and bounces off.
	Mass float64
	// Restitution is how much of the approach speed a contact gives back, from
	// 0, perfectly inelastic, to 1, perfectly elastic. A pair bounces by the
	// lower of its two sides.
	Restitution float64
}

// DefaultMass is what an entity whose Physics names no Mass weighs — the value
// that makes an equal pair exchange velocities.
const DefaultMass = 1.0

// Weight is what this entity weighs in a collision — DefaultMass unless Mass is a real weight.
func (p Physics) Weight() float64 {
	if p.Mass <= 0 {
		return DefaultMass
	}
	return p.Mass
}

// Bounce is Restitution held within its range.
func (p Physics) Bounce() float64 { return min(max(p.Restitution, 0), 1) }

// Immovable reports whether nothing can shift this entity.
func (p Physics) Immovable() bool { return math.IsInf(p.Mass, 1) }
