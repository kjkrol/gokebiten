// Package vision gives entities a narrowed view of the world: a Sight cone sees what falls
// inside it, within range and not hidden behind something nearer. Each tick fills Sight.Seen
// and runs Between behaviors of a Sighting; SightOutline gets the view drawn.
//
// # Sight and Seen
//
// [Sight] is what an entity can take in — Facing, its own direction whichever way it moves;
// HalfAngle either side of it; Radius — and what the last scan found there: [Sighted], at most
// [MaxSeen] entities nearest first. The outline buffer is sized for [MaxSightRadius] and
// MaxHalfAngleMilli; a larger Sight still sees, its outline is only coarser. The [ScanSystem]
// scans every Sight against the world's space through aabbworld's line-of-sight scan, once a tick.
//
// # Transparency
//
// Every entity cuts sight unless it carries a [Transparency]: 1 as if absent, 0 cutting, in
// between dimming — a ray spends its Radius as a budget and a stretch through an entity at τ costs
// 1/τ per unit, so a forest at 0.4 is looked through at 0.4 of the reach. The board gives its
// veiled terrain bodies one (1 - Veil). A Sight with Clear set looks over whatever only dims — a
// flyer over the forest — and is still cut by what cuts.
//
// # Sighting
//
// A [Between] behavior registered here is run once a tick per observer carrying tag a,
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
