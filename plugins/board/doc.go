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
// its movement cost, the [Domain]s it admits, whether it is solid (a wall) or opaque (a forest:
// sight stops at it), and the sprite drawn for it; kinds are created
// through the Plugin's [CellKindDict]. Cost 1 is full speed and the baseline path weight; above 1
// slows and costs more to plan through; below 1 is a boost a game may choose to offer. [Terrain] is what a cell answers about itself.
//
// # Domains, Mover and Standing
//
// A [Domain] is a way of moving — [Land], [Water], [Air], or a game's own bit — and a cell's
// Allows says which may stand on it: water admits Water, a hole nobody. An entity's [Mover] says
// which it uses (none means Land); the planner keeps it to cells that admit it. Every tick, after
// collisions, the board tells each entity carrying Cell where it stands as a [Standing] —
// [Plugin.RegisterBehavior] takes a plugin.Each of it, naturally one over Mover — and
// [Standing.Fell] says the entity is where its domain may not be: pushed into water, dropped
// into a hole. What follows is the
// game's: despawn, teleport, damage. Call [Plugin.RunPlan] every tick, after collision's.
//
// # Terrain bodies
//
// Built [Plugin.WithCollision], the board makes its Solid terrain physical: every run of solid
// cells becomes an immovable entity in the world — a [Body] with a collider and an
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
