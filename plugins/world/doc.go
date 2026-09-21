// Package world is the foundation for a Stage with moving, drawable entities: a Position and
// Velocity each, a shared spatial index other plugins query, and motion integrated each tick,
// never further than Position.MaxStep. What an entity carries is set by its kind — see kind.
package world
