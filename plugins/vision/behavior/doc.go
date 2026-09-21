// Package behavior holds ready-made reactions to what an entity sees, each a plain function
// of a vision.Sighting handed to plugin.Between: Flee gives way to what is on a collision
// course and runs from a Threat, Chase goes after the nearest and searches when it sees none.
package behavior
