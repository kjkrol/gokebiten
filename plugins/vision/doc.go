// Package vision gives entities a narrowed view of the world: a Sight cone sees what falls
// inside it, within range and not hidden behind something nearer. Each tick fills Sight.Seen
// and runs plugin.Between behaviors of a Sighting; SightOutline gets the view drawn.
//
// # Sight and Seen
//
// [Sight] is what an entity can take in — Facing, its own direction whichever way it moves;
// HalfAngle either side of it; Radius — and what the last scan found there: [Sighted], at most
// [MaxSeen] entities nearest first. The radius and the half-angle are capped ([MaxSightRadius],
// MaxHalfAngleMilli), which bounds the outline buffer. The [ScanSystem] scans every Sight against
// the world's space through aabbworld's line-of-sight scan, once a tick.
//
// # Sighting
//
// A plugin.Between behavior registered here is run once a tick per observer carrying tag a,
// with a [Sighting]: the observer, its Base, Sight and Steering (nil for one that cannot be
// steered), and everything in view carrying b as [Seen] values nearest first — a directed pair,
// grouped by observer, run even when nothing is in view. A behavior tells its seen entities
// apart with Seen.Carries, and steers only through Steering.Request. Ready-made
// ones, and their tags, are in plugins/vision/behavior.
//
// # SightOutline and Renderer
//
// An entity also carrying [SightOutline] has its view's shape computed: a reach per evenly spaced
// angle across the cone. The [Renderer] draws it through the camera in a [ConeStyle]
// ([DefaultConeStyle] strokes the boundary; [Plugin.WithStyle] or [ConeStyleFn] for another).
package vision
