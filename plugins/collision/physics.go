package collision

import "math"

// Physics makes an entity take part in the physical side of a contact: pushed apart, bouncing.
// The zero value weighs DefaultMass and does not bounce.
type Physics struct {
	// Mass is how hard the entity is to shift; non-positive weighs DefaultMass, +Inf is a wall.
	Mass float64
	// Restitution is the share of approach speed given back, 0 to 1; a pair uses the lower one.
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
