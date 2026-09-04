// Package board lays a grid (square or hex) over the game world, with
// per-cell terrain (passable/impassable, movement cost, sprite) and
// occupancy tracking. It publishes a world.SpeedModifier so entities on
// the board automatically move at the terrain's cost - plugins/navigation
// builds pathfinding on top of it.
package board
