// Package world is the mandatory foundation for any game with moving,
// drawable entities. It gives every entity a Position and Velocity, keeps
// them in a shared spatial index other plugins can query, and integrates
// motion each tick - SpeedModifiers (e.g. terrain cost) scale it, and
// displacement is capped per tick so fast entities can't tunnel through
// each other.
package world
