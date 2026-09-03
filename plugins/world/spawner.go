package world

import "github.com/kjkrol/uid"

// Spawner builds one World.Populate call: Position and Velocity are
// required, bound at construction; every other component is attached via
// With/WithEffect.
type Spawner struct {
	position func(index, count int) Position
	velocity func(index int) Velocity
	extras   []entityExtras
}

// NewSpawner requires the two components every entity must have —
// Position (computed from index and the total count, for grid-style
// layouts) and Velocity (computed from index alone).
func NewSpawner(
	position func(index, count int) Position,
	velocity func(index int) Velocity,
) *Spawner {
	return &Spawner{position: position, velocity: velocity}
}

// With attaches a further component, set from index alone.
func (s *Spawner) With[T any](value func(index int) T) *Spawner {
	s.extras = append(s.extras, newComponentAdder(value))
	return s
}

// WithEffect attaches a further component plus a side effect run right
// after it's written — for state (an occupancy tracker, say) that must
// react to the value as it's set, not just store it.
func (s *Spawner) WithEffect[T any](value func(index int) T, effect func(v T, id uid.UID64)) *Spawner {
	s.extras = append(s.extras, newComponentAdder(value).WithEffect(effect))
	return s
}
