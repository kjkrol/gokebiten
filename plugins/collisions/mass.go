package collisions

// Mass is how hard an entity is to shift in a collision, in arbitrary units.
// An entity without one — or with a non-positive Value — weighs DefaultMass.
type Mass struct {
	Value float64
}

// DefaultMass is what an entity with no Mass weighs: the value that makes a pair
// exchange velocities, as every entity did before Mass existed.
const DefaultMass = 1.0

// Weight is what this entity weighs in a collision — DefaultMass unless Value is a real weight.
func (m Mass) Weight() float64 {
	if m.Value <= 0 {
		return DefaultMass
	}
	return m.Value
}

// massOf is what m weighs, for a side that may carry no Mass at all.
func massOf(m *Mass) float64 {
	if m == nil {
		return DefaultMass
	}
	return m.Weight()
}
