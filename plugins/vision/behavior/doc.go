// Package behavior holds ready-made reactions to what an entity sees, each a plain function
// of a vision.Sighting handed to plugin.Between: Flee gives way to what is on a collision
// course and runs from a Threat, Chase goes after the nearest and searches when it sees none.
//
// # Flee
//
// [Flee] steers the observer away from whatever is closing on it, and from any [Threat] the
// moment it comes into view; register Flee.Steer between Skittish and plugin.Any; the tags come from [DefineTags]. [Flee.SetEnabled]
// switches it off and on without unregistering.
//
// # Chase
//
// [Chase] steers at the nearest entity the observer is shown; shown none, it turns a quarter
// aside every lookEvery, searching.
//
// # Tags
//
// [Predator] and [Prey] are ready-made tags for the two sides of a hunt, [Skittish] for the
// entities Flee steers, [Threat] for what is fled on sight; any tag of the game's own does as
// well. Who flees or hunts whom is the registration's to say, and a file using both plugins'
// behaviors imports them as cbehavior and vbehavior.
package behavior
