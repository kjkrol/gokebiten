// Package board lays a square or hex grid over the game world, with per-cell terrain
// (passability, movement cost, sprite) and occupancy tracking. Entities on the board
// move at the terrain's cost; plugins/navigation builds pathfinding on top.
package board
