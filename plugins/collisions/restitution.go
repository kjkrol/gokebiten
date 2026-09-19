package collisions

// Restitution is how much of the approach speed a contact gives back, from 1
// (bounces fully) to 0 (not at all). An entity without one bounces fully.
type Restitution struct {
	Value float64
}

// FullRestitution is what an entity with no Restitution bounces with.
const FullRestitution = 1.0

// restitutionOf is r's bounce held within [0,1], or FullRestitution when there is none to read.
func restitutionOf(r *Restitution) float64 {
	if r == nil {
		return FullRestitution
	}
	return min(max(r.Value, 0), FullRestitution)
}
