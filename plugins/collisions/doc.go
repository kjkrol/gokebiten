// Package collisions detects and resolves overlaps between world entities
// each tick, over world's shared spatial index, and publishes what each entity
// struck in Contacts — Sensor entities detect without pushing, Static entities
// never move.
//
// How hard a contact hits back is the entities' own business: Mass and
// Restitution shape the impulse it publishes, each falling back to a sensible
// default when absent. Neither touches the push that separates the overlap — that one always splits the penetration evenly between the two
// sides, whatever they weigh.
//
// Reacting to a collision is a world.Behavior reading Contacts — "who I struck
// and how hard" — registered with world.Plugin.RegisterBehavior. That goes for
// the bounce itself (strategies/elastic) as much as for showing a hit
// (strategies/hit), counting (strategies/stats) or logging (strategies/debug):
// this package detects and separates, and every reaction to that is a
// behaviour a game registers. Those run once per tick, in their own phase,
// and can add or remove components freely, which is what "the bullet is gone
// and the target has lost HP" needs. Which entities a reaction applies to is
// the behavior's own query: a tag given to a kind scopes it to that kind, a
// tag given through one roster entry scopes it to one entity.
//
// A reaction reads the contacts of the tick that has just finished, since the
// collision phase runs after the behaviors do. Only entities the broad phase
// walks over — the ones carrying Velocity — record contacts; a Static wall
// never does, having nothing to clear its list each tick.
package collisions
