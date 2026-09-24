# Changelog

## Unreleased

Saves written by v0.2.0 do not load: `Base` and the marker components changed shape.

**Movement**
- `world.Steering` holds a motion profile — `MaxSpeed`, `Accel`, `Brake`, `V0`, `TurnRate` — and
  `SteeringSystem` writes the base speed every tick; navigation steers through it: a lookahead
  point, waypoints passed by projection, braking to rest on the goal, a queue of goals
  (Shift + right click), routes previewed to every queued goal.
- Routes and legs lose their footing when the terrain changes under them; a unit stuck where its
  domain may not keeps its order.
- The camera pans in screen pixels at any zoom.

**Board**
- Terrain kinds say whom they admit (`Allows`, a bitset of `Domain`s), whether they are `Solid`,
  how much they `Veil` sight (0 clear, 1 cutting; a forest 0.6), and what they cost per domain
  (`Costing`, `CostFor`); a unit's `Mover` says how it moves. `Passable` is gone.
- `WithCollision`: solid terrain becomes immovable bodies built from `Grid.CellBoxes` — one box
  for a square, capped strips for a hex — merged up to `MaxBodyCells` a side; veiled terrain
  becomes bodies carrying a `vision.Transparency` of 1 - Veil, dimming sight only.
- `Standing`, reported every tick to `plugin.Each` behaviors: the cell under an entity, its kind,
  its box; `Fell(domain)` says the entity is where it may not be.
- `Grid.CellsUnder`, `CellBounds`, `CellOutline`; hex cells drawn as hexagons; `TerrainMap.Version`.
- Cell entities (`CellEntity`, `Ground`, `WithEffects`) let an effect change terrain for a while.

**Plugins**
- Tags are bits of families: `plugin.Tags[F]` is one component per family, `Kinds.DefineTag`
  names the bits (saved by name), `kind.Tagged` gives them to a kind, `Between(a, b, fn)` takes
  them as values; `Selectable` and `Selected`, the vision behaviors' tags and terrain bodies are
  bits. `navigation.NewPlugin` takes the selection plugin.
- `plugins/effects`: temporary changes to entities — `Grant` and `Alter` in a `Spec`, `Lasts`
  or until `Dispel`, `Cast`/`CastFor`/`Dispel`/`Has` by entity id, `Active` saved with the entity.
- `plugins/world`: `Kinds.Reserve` and `Bodies` for kind-less entities; `Kinds.DefineTag`.
- `collision.Detector` is `CollisionSystem` (`NewCollisionSystem`), as every system is named.
- `vision`: sight through terrain — an entity carrying `Transparency` dims sight instead of
  cutting it (aabbworld v1.6.0: a ray spends its radius as a budget, a forest at 0.6 takes 2.5×
  its depth), `Sight.Clear` looks over the veils (a flyer), what cuts sight still cuts. A
  `Sight.Radius` above `MaxSightRadius` works; only the outline is coarser.
- Flying is a convention, not a feature: `Mover{Domain: Air}` on kinds that admit `Air`, a
  `Collider` without `Physics` so nothing pushes the flyer, `Costing(Air, 1)`, `Sight{Clear: true}`.
- `navigation`: route arrows every 15°, so hex steps draw true.
- `render`: `Hexagon`; `QuadBatch` draws in chunks under the 16-bit index limit.

**Demos**
- `navigation-hex-demo`, `navigation-vision-demo`, `navigation-vision-hex-demo`, `island-demo`,
  `effect-demo`. The two vision demos have a hawk that flies over the wall and the forest and sees
  through the forest, whose veil fades the other units' cones.

## v0.2.0 — 2026-09-22

Renamed to **gram**: the module is `github.com/kjkrol/gram`, the root package `gram` (`gram.Run`).
Plugin resource names (`gram.world`, `gram.collision`, ...) follow, so saves written by gokebiten
do not load. The API below is what settled since v0.1.1.

**Game**
- Stages and Scenes: a `game.Game` is a named set of `game.Stage` values, each with its own ECS
  built when entered, and each Stage a set of `game.Scene`s with a live `Composition` saying what
  is shown and which Scene takes input. `game.Runtime` is one interface for pause, quit, switching
  Stage, persistence and the camera.
- A launcher: `gram.Run(game)`.
- Persistence: resources matched by name; `PostLoader` and `Restorer` hooks.

**Plugins**
- `plugin.Plugin` is the one extension point; behaviors (`plugin.Between`, `plugin.Each`) are
  registered on the plugin that hosts them and refused elsewhere.
- `plugins/world`: config moved into the plugin (`ctx.UseWorld`); edge rules per axis; `OnExit`
  and `Tracked`; `Attach`, `Detach`, `Declare`; the entity renderer with appearance modifiers.
- `plugins/world/kind`: kinds defined from a `Spec` of `Const` and `Load` components; `Seed` and
  `Populate`.
- `plugins/collision`: `Collider` and `Physics`, `ShapeTest`, `Meeting` and `Struck` behaviors,
  ready-made `CountContacts`, `ShowHits`, `LogContacts`.
- `plugins/vision`: `Sight` cones scanned each tick into `Seen`, `Sighting` behaviors, drawn
  outlines; ready-made `Flee` and `Chase`.
- `plugins/board`: single-occupancy fix. `camera`: a root package with zoom limits.
- Clean architecture: `game`, `plugin` and `render` public; only `internal/engine` internal.
- On aabbworld v1.5.0: collisions through `collide.Engine`.

**Project**
- Every package has a `doc.go`; the root one carries the concepts, the tick lifecycle and the
  package graph. README rewritten, with `examples/minimal` as its example.
- All benchmarks in `bench/`; `Makefile` with `bench` and `bench-save`; results and method in
  `BENCHMARKS.md`.

## v0.1.1

Resources reorganised, camera improvements, internal logic hidden behind a public API.

## v0.1.0

First release: `Game`, `Plugin`, world, physics and camera plugins, the collision demo.
