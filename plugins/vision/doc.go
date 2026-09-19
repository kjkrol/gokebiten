// Package vision gives entities a narrowed view of the world: an entity with a
// Sight cone sees only what falls inside it, within range, and is not hidden
// behind something nearer.
//
// It publishes facts and decides nothing. Each tick the scan writes Sighted —
// who this entity can see, nearest first — and any Behavior registered with
// world.RegisterBehavior reads it and acts. An entity that also carries
// SightOutline gets the drawn shape of its view computed, which Renderer turns
// into a fan on screen.
//
// A cone reaching past the edge of a toroidal world wraps the same way a sprite
// does: the outline is drawn once per image of the world it reaches into, off
// the fragments plane.AABB already carries.
package vision
