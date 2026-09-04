// Package collisions detects and resolves overlaps between world entities
// each tick, over world's shared spatial index. It tags colliding entities
// with Hit and dispatches to pluggable CollisionHandlers (e.g. elastic
// bounce, or your own) - Sensor entities detect without pushing, Static
// entities never move.
package collisions
