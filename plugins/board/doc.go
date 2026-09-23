// Package board lays a square or hex grid over the game world, with per-cell terrain
// (passability, movement cost, sprite) and occupancy tracking. Entities on the board
// move at the terrain's cost, impassable terrain can be made solid, and plugins/navigation builds
// pathfinding on top.
//
// # Board, Grid and Layout
//
// A [Grid] is a topology behind neighbor, coordinate and distance queries; [DefaultGrids] makes a
// square or a hex one, and each wraps per axis following the world's edges. A [Board] pairs a
// Grid with its [TerrainMap], the one place to read the topology and read or write terrain.
// [Plugin], built over a Grid, an [Occupancy] and the world plugin, seeds its terrain from a
// [Layout] (a default kind for every cell, then per-cell overrides) when the Stage starts fresh,
// saves it, and slows the world's entities by the terrain they stand on
// ([TerrainSpeedModifier]).
//
// # Cell, CellKind and Terrain
//
// A [CellID] names one cell; [Cell] is an entity's current one. A [CellKind] is a named terrain:
// its movement cost, whether it is passable, whether it is opaque (a forest: passable, but sight
// stops at it), and the sprite drawn for it; kinds are created
// through the Plugin's [CellKindDict]. Cost 1 is full speed and the baseline path weight; above 1
// slows and costs more to plan through; below 1 is a boost a game may choose to offer. [Terrain] is what a cell answers about itself.
//
// # Terrain bodies
//
// Built [Plugin.WithCollision], the board makes its impassable terrain solid: every run of
// impassable cells becomes an immovable entity in the world — a [Body] with a collider and an
// infinite mass, no sprite, no kind — so no unit ends a tick inside a wall and walls occlude sight;
// a run of opaque cells becomes a Body without a collider, occluding only.
// A body is made of the boxes the grid gives for each cell ([Grid.CellBoxes]: one for a square,
// [HexCapStrips] strips over each cap of a hex, covering it from outside), merged along both axes
// up to [MaxBodyCells] a side. The bodies follow [TerrainMap.Version]; call [Plugin.RunPlan] after
// collision's.
//
// # Occupancy
//
// [Occupancy] tracks who holds each cell, gating and recording every step navigation takes:
// [SingleOccupancy] lets one entity in, [MultipleOccupancy] any number.
//
// # Renderer
//
// [Plugin.WithRenderer] builds the [Renderer] drawing each cell's sprite from an atlas; put it
// under the entity layer. [RenderState] holds its live toggles, such as grid lines.
package board
