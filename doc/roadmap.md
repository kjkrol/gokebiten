# Roadmap

[← Back to README](../README.md)

Where the engine stands and what comes next, in the order it is meant to come. The reasoning
behind each item lives in [movement.md](movement.md) and [views.md](views.md).

## Done

- Movement through a motion profile: `world.Steering` with `V0`, `Accel`, `Brake` and
  `TurnRate`; navigation asks for headings at a lookahead point, passes waypoints by projection,
  brakes to rest on the goal, queues goals (Shift + right click) — [movement §1–§4, §7](movement.md).
- Terrain as the one truth of the board: solid cells are bodies built from the boxes of any grid
  (square and hex), domains say who may stand where (`Allows`, `Solid`, `Opaque`, `Mover`), the
  board reports where everyone stands (`Standing`, `Fell`), costs are priced per domain, and
  routes are checked whenever the terrain changes — [movement §5, §6, §11](movement.md).
- Tags as bits of families, one component per family; `Between(a, b, fn)` by value;
  `Selectable`/`Selected`, the vision behaviors' tags and terrain bodies on bits.
- Effects: `Grant` and `Alter` in a `Spec`, cast by entity id, saved with the entity; cell
  entities with `Ground` so an effect can change terrain for a while — [movement §13](movement.md).
- Hex boards on screen, route arrows every 15°, camera panning in screen pixels, a `QuadBatch`
  that draws in chunks; six demos, `effect-demo` among them.

## Next

1. **Sight through terrain, sight range, flying** — transparency per `CellKind`, attenuation in
   aabbworld's raycast, `MaxSightRadius` into configuration, `Mover{Air}` with collision's veto
   against solid bodies — [movement §12](movement.md).
2. **Players** — a plugin: a view, a command queue and a translator per player; command types and
   labelled default bindings shipped by the plugins; one local player first, split screen after —
   [views §2](views.md).
3. **Hover** — what is under the cursor, a `Space.Query` at a point in the translator — [views §2](views.md).
4. **`RouteStyle`** — `CellArrows` by default, `SmoothRoute` opt-in, arcs from the profile,
   computed when the route changes — [movement §8](movement.md).
5. **Effects over the whole board** — weather and seasons as effects on an entity standing for
   the board — [movement §13](movement.md).
6. **Networking** — `netview` over players: deltas from the `View`, one mask per client, a frame
   a tick, a client without an ECS — [views §3](views.md).
7. **Turn-based movement** and **arbitration** — when a game needs them — [movement §9, §10](movement.md).

Also on the list: `CellKind` without a string in `Ground` (goke warns about the locality of a
component holding one); saves written before tag families do not load, to be noted at the next
tag; that tag, v0.3.0, once this state has been reviewed.
