// Package collisions detects overlaps between world entities each tick, over
// world's shared spatial index, and publishes what each entity struck on its
// Collision.
//
// Whether an overlap is more than detected is the entities' own business. One
// carrying Physics takes part in the physical world: it is pushed out of what
// it overlaps and bounces off it, by its Mass and Restitution, with an infinite
// Mass making it a wall nothing shifts. One without Physics is only ever
// detected — a town to walk into, a trigger, a hunter's jaws — and a pair is
// detect-only as soon as either side is. The push that separates two physical
// entities always splits the penetration evenly, whatever they weigh; only the
// bounce is weighted.
//
// The bounce is the engine's own, settled contact by contact as the pairs are
// resolved. Everything else that should happen on a contact is a behavior
// hosted by this plugin — built with plugin.Between or plugin.Each, registered
// with Plugin.RegisterBehavior, and run inside the engine's own passes, so none
// of them costs a query of its own:
//
//   - plugin.Between[A, B] given a func of Meeting runs for every contact
//     between an entity carrying A and one carrying B, in the narrow phase. A
//     and B join its query as optional components, plugin.Anything stands for
//     whatever is there: "the bullet is gone and the target has lost HP" is
//     Between[Bullet, Target].
//   - plugin.Each[T] given a func of Struck runs once a tick on every collidable
//     entity carrying T, in the broad phase, with what it struck the tick before.
//
// Meeting and Struck are what make a behavior this plugin's: one built for
// another host's payload is refused.
//
// Ready-made ones live under strategies: hit, stats, debug. Each exports a
// plain function, and the tags it runs between are named where it is
// registered — RegisterBehavior(plugin.Between[Bullet, Target](debug.Log())).
// A behavior that should apply to one kind gives that kind its tag; one that
// should apply to a single entity gives the tag through that roster entry.
//
// Collision.Contacts is the same information kept on the entity, for anything
// that would rather read it than be called: who was struck, how hard, and which
// way this entity left. It holds the contacts of the tick that has just finished.
package collisions
