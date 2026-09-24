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
// saves it, and slows every entity carrying a [Mover] by the terrain under it (a Moving behavior
// it registers on the world).
//
// # Cell, CellKind and Terrain
//
// A [CellID] names one cell; [Cell] is an entity's current one. A [CellKind] is a named terrain:
// its movement cost, the [Domain]s it admits, whether it is solid (a wall) or how much it veils
// sight (a forest), and the sprite drawn for it; kinds are created
// through the Plugin's [CellKindDict]. Cost 1 is full speed and the baseline path weight; above 1
// slows and costs more to plan through; below 1 is a boost a game may choose to offer.
// [CellKind.Costing] prices a kind differently for some domains — elves through a forest, a
// witch over snow — and [CellKind.CostFor] is what an entity pays: the cheapest of its domains
// the kind admits and prices, else Cost. [Terrain] is what a cell answers about itself.
//
// # Domains, Mover and Standing
//
// A [Domain] is a way of moving — [Land], [Water], [Air], or a game's own bit — and a cell's
// Allows says which may stand on it: water admits Water, a hole nobody. An entity's [Mover] says
// which it uses (none means Land); the planner keeps it to cells that admit it. Every tick, after
// collisions, the board tells each entity carrying Cell where it stands as a [Standing] —
// [Plugin.RegisterBehavior] takes an [Each] of it, naturally one over Mover — and
// [Standing.Fell] says the entity is where its domain may not be: pushed into water, dropped
// into a hole. What follows is the
// game's: despawn, teleport, damage. Call [Plugin.RunPlan] every tick, after collision's.
//
// # Cell entities and Ground
//
// [Plugin.CellEntity] gives a cell an entity — a body with a [Cell] and a [Ground] holding the
// cell's kind — so anything done to entities can be done to a cell: an effect altering Ground is
// a temporary change of terrain. While the entity exists the board copies its Ground into the
// TerrainMap every tick; [Plugin.DropCellEntity] lets it go, and the board does that itself once
// the entity's last effect ended (effects.Idle on it, no Active).
//
// # Terrain bodies
//
// Built [Plugin.WithCollision], the board makes its Solid terrain physical: every run of solid
// cells becomes an immovable entity in the world — tagged [Plugin.Body] in board's tag [Family], with a collider and an
// infinite mass, no sprite, no kind — so no unit ends a tick inside a wall and walls cut sight;
// a run of veiled cells becomes a body without a collider carrying a vision.Transparency of
// 1 - Veil, so a cone fades through it, on the world.Layers of the kind's Veils, so an observer
// whose Sight.Blockers miss them looks over it.
// A body is made of the boxes the grid gives for each cell ([Grid.CellBoxes]: one for a square,
// [HexCapStrips] strips over each cap of a hex, covering it from outside), merged along both axes
// up to [MaxBodyCells] a side. The bodies follow [TerrainMap.Version]; call [Plugin.RunPlan] after
// collision's.
//
// The board requires of every unit a [Cell] (where it starts) and a [Mover] (the domains it moves
// in) through the world's kind.Roster — and makes them itself in [Units]: a game binds its rows to
// the board once ([NewUnits]: the units' [Shape], where a row says a unit stands) and defines each
// kind by its Mover and steering profile plus its own components; Position and Cell come from the
// one point, Layers from the domain.
//
// # Heights
//
// In a Quasi3D world (world.Config.Quasi3D) a [CellKind] has an Altitude, its ground level, and a
// Height, what stands on it. The [Board] keeps a raster of altitudes, one per cell (Grid.Ordinal),
// rebuilt when the terrain's Version moves, and is the world's Ground ([Board.GroundAt],
// [Board.Step]). Every tick the board writes each Z-carrying entity's Altitude: the ground under
// its centre plus its Mover's Lift, so a unit never declares where it stands in height and a hawk
// declares only how high it flies. Units get their Z from the Shape, terrain bodies from their
// kind. A hill is a number in the raster and never a body, so the cost of sight does not depend
// on how many a game has. On a square grid the ground runs smoothly between cells: each corner
// stands at the mean altitude of the cells that meet there ([Board.Corners]), GroundAt reads
// between a cell's corners, a hill has slopes and a unit on a slope stands at its height; the
// renderer draws the tiles sloped and faces only where a top stands above its neighbour's — a
// wall over grass, a raised edge over the sea. A flat world refuses an Altitude, a Height or a
// Lift where it first meets one.
//
// # Occupancy
//
// [Occupancy] tracks who holds each cell and in which domains, gating and recording every step
// navigation takes: [SingleOccupancy] lets one entity per domain into a cell (a walker and a
// hawk share one, two walkers do not), [MultipleOccupancy] any number — tokens on a square, which
// carry no Physics, since bodies cannot overlap. Solid terrain bodies are on the world.Layers of
// whoever their kind keeps out, so a wall admitting Air lets a flyer over and cuts none of its
// sight.
//
// # Renderer
//
// [Plugin.WithRenderer] builds the [Renderer] drawing each cell's sprite from an atlas; put it
// under the entity layer, or into a render.Sorted with the world's renderer. There it submits each
// cell's top at its altitude plus its kind's Height and, through an isometric camera, the two faces
// towards the viewer wherever the ground drops to a neighbour (a cliff, down to sea level 0 off the
// board) or the kind stands tall (a wall), shaded as if lit from the upper left. [RenderState] holds
// its live toggles, such as grid lines (drawn only in the plain layer).
package board
