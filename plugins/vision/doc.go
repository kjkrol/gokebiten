// Package vision gives entities a narrowed view of the world: an entity with a
// Sight cone sees only what falls inside it, within range, and is not hidden
// behind something nearer.
//
// It publishes facts and decides nothing. Each tick the scan writes Sight.Seen —
// who this entity can see, nearest first — and hands it to the behaviors
// registered with Plugin.RegisterBehavior, inside the scan's own pass: a
// plugin.Between[A, B] given a func of Sighting runs once a tick for every
// observer carrying A, with everything in its view that carries B, nearest
// first — and with nothing, when there is nothing of the kind to see. A sighting
// has a direction, so Between[Predator, Prey] is never the prey looking back.
// The scan looks each seen entity up once for all of them, so a behavior costs
// no query of its own; one that needs to tell its seen entities apart declares
// the tag with plugin.Asking and asks Seen.Carries. Ready-made ones live under
// strategies: flee, hunt — plain functions, so who flees or hunts whom is named
// where they are registered. Changing an observer's course goes through its
// Steering.Request.
//
// An entity that also carries SightOutline gets the drawn shape of its view
// computed, which Renderer turns into a fan on screen.
//
// A cone reaching past the edge of a toroidal world wraps the same way a sprite
// does: the outline is drawn once per image of the world it reaches into, off
// the fragments plane.AABB already carries.
package vision
