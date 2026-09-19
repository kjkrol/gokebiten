// Package collisions detects and resolves overlaps between world entities
// each tick, over world's shared spatial index. It tags colliding entities
// with Hit and dispatches to pluggable CollisionHandlers (e.g. elastic
// bounce, or your own) - Sensor entities detect without pushing, Static
// entities never move.
//
// A CollisionHandler is a hook inside the solver, not a place to put game
// behaviour: it runs on live Position/Velocity pointers, possibly several times
// per tick, and what it writes feeds the separation happening around it. That
// is what elastic needs and what nothing else should want.
//
// To react to a collision rather than resolve it, read the components this
// package publishes — Hit for "something struck me", Collision.Touching for
// "who I am against right now" — from a world.Behavior registered with
// world.Plugin.RegisterBehavior. Those run once per tick, in their own phase,
// and can add or remove components freely.
package collisions
