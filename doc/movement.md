# Movement: along a path, through steering, past walls and holes

[← Back to README](../README.md) · [Views, players and commands](views.md)

> A design sketch, not a contract: how a unit should move along a planned route, what it is
> made of, and what happens when the world pushes it somewhere it may not go. Nothing here is
> code yet.

## Today, and why it looks the way it looks

`navigation`'s system writes straight into `Base.Vel.Dir` and `Vel.Value`, aiming at the
**centre** of the next cell. Arrival is being within `arrivalEpsilon` (2 world units) of that
centre; then the speed drops to zero, the box is nudged onto the centre, the `Leg` is released,
and the next waypoint is aimed at. `world.Steering` — `Want`, `Pending`, `TurnRate`, `Reflex` — and
the `SteeringSystem` that turns a heading by at most `TurnRate` a tick take no part in it. That is
the whole reason a unit arrives, stops, turns and sets off again: nothing ever asked it to turn.

Two things already work and stay: a unit knocked off its `Leg` (its box's centre outside the leg's
cells) releases the leg, drops its `Path`, and re-plans next tick; a unit whose next cell is taken
waits (`targetWaitTimeout`) and then settles for the `nearestFree` cell.

## 1. One actuator: Steering

The only thing that moves a unit — turns it and speeds it up — is `world.Steering`. Everything
else asks: `Steering.Request(dir)` for a heading, `Steering.RequestSpeed(v)` for a speed. Navigation
stops writing `Vel.Dir` and `Vel.Value`. The profile is active when `MaxSpeed` is above zero;
without one, `Steering` steers headings only and `Vel.Value` stays whoever's it was — which is
what every unit that carries `Steering` today gets.

This settles the levels of the movement commands from [views.md](views.md): `MoveTo{Cell}` is a
goal for the planner (navigation), `Steer{Dir}` is a reactive intention (flee, chase); both end in
a `Request`. An AI may speak at either level. Speed is the unit's own: navigation asks for its
`MaxSpeed` and brakes from its `Accel`, and `navigation.NewPlugin` takes no speed — unit types
differ in tempo through their kind. A unit with a `MoveOrder` but no `Steering` profile is not
navigated.

### The motion profile lives in Steering

Beside `TurnRate` and `Reflex`, `Steering` carries how the unit accelerates:

| Field | Meaning |
|:---|:---|
| `MaxSpeed` | the unit's top base speed, world units a second; zero means no profile. A speed past the step `StepReach` allows (`Position.MaxSpeed(tps)`) is clipped by the move, tick by tick — a kind does not know the tick rate, so nothing refuses it earlier |
| `Accel` | how fast it speeds up and slows down, units a second²; a separate brake only when a game asks for one |
| `V0` | the speed it has the instant it sets off from standing: a walker walks at once, a tank starts from nothing |
| `Speed`, `WantSpeed` | state: the current base speed, and the one asked for |

Each tick the `SteeringSystem` turns `Vel.Dir` towards `Want` by at most `TurnRate`; from
standing it jumps to `min(V0, WantSpeed)`, then moves `Speed` towards `WantSpeed` by `Accel·dt`,
never past `MaxSpeed`; and it **writes `Vel.Value = Speed` afresh every tick**, before the
`VelocitySystem` scales it by the speed modifiers (terrain). That last point repairs a fragility
of today: `VelocitySystem` multiplies `Vel.Value` in place, so whoever owns the base speed must
rewrite it every tick or the modifiers compound — navigation happens to, other units happen not to.

Everything below follows from the profile. The turning radius is `Speed·dt / TurnRate`: wide arcs
at full speed, tight ones when setting off. Braking follows `v = sqrt(2·Accel·d)`: navigation asks
for that speed at distance `d` from the goal, never below the speed braking leaves at the arrival
radius, and the unit comes to rest on the goal. The profile is
the kind's (`kind.Const(world.Steering{MaxSpeed: 120, Accel: 200, V0: 40, TurnRate: 0.1})`), so
unit types differ in how they move without any code.

## 2. Where to look: the lookahead point

Instead of the centre of the current cell, the unit aims at a point on its path a distance
`reach = Speed·dt / TurnRate` ahead — the turning radius: the way covered in a tick over the radians
turned in a tick. Walk the path's segments from the
unit's projection onto them, lay off `d`, take the point (on a torus through `shortestAxisDelta`,
as today). As the point passes a bend in the path, the requested heading starts to rotate before
the unit reaches the cell, and the `SteeringSystem` carries it round the arc. With `TurnRate = 0`
the point is the next waypoint and the unit moves as it does today.

Speed scales with `max(cos θ, 0.2)` of the angle between the heading and the requested
direction: straight on keeps full speed, a sharp bend slows the unit, a U-turn crawls — so the
turning radius shrinks with the turn and a unit turns round almost on the spot. Slowing to a
halt happens only at the end: `WantSpeed = 0` from the braking distance, then the nudge onto the
goal's centre as today.

## 3. Passing a waypoint is not being near its centre

A waypoint is passed when the unit's projection onto the waypoint's segment goes beyond the
segment's length — it has crossed the plane perpendicular to the segment at the waypoint — or
when the unit is within the lookahead reach of it, since from there the lookahead already looks
past it. Without the second rule a route that folds back at the waypoint keeps the unit circling
in front of the plane forever. It never has to touch the centre. `arrivalEpsilon` remains for the
final goal only.

A new route that goes back the way the current leg came turns the leg round on the spot — the
same cells are held — so a unit ordered back does not first finish the step it was on.

## 4. Entering a cell is its own event

Entering a cell — for `Occupancy` and the `CellEntered` tag — is `CellAt(centre of the box)`
changing, as today, and has nothing to do with waypoints. The next cell is reserved (`Leg`) ahead
of time, when the lookahead point enters it; if `CanEnter` refuses, the unit asks for
`WantSpeed = 0` and waits (`targetWaitTimeout`, `nearestFree` as today) — the one legitimate halt
on a route. `Reflex` delays a reaction, it does not stop anything: requests come every tick and
`Steering` coalesces them.

## 5. Obstacles come from two sources and must count both ways

Impassable cells of the board are a physical obstacle (walls, below). Static collider entities — a
building, a rock: `Collider` plus `Physics{Mass: +Inf}`, whether or not the board knows them — are
an obstacle for the planner: the cells their box covers are taken in `Occupancy` and impassable
for `findPath`, entered when such an entity spawns or moves and released when it goes. One map of
obstacles; neither side sees only its own.

## 6. Pushed onto forbidden ground

What happens when a collision pushes a unit towards a cell it may not enter is the cell kind's
decision (`board.CellKind`). Walls are done; holes and sight through terrain are open.

- **A wall — done.** Impassable cells are **terrain bodies**: entities with a `Base`, a
  `collision.Collider`, a `collision.Physics{Mass: +Inf}` and the `board.Body` tag, no
  `Appearance` (the board draws the cell), no kind (`world.Kinds.Reserve`, spawned through
  `world.Bodies`). A body is made of boxes: each grid says what boxes cover one of its cells
  (`Grid.CellBoxes`) — a square is one box, a hex is a middle band plus `HexCapStrips` strips over
  each cap, as wide as the hex is at the strip's wider edge, so the cover is never smaller than
  the cell; an irregular region (a province) will give the boxes of its raster. Touching boxes of
  one kind are merged into rectangles, at most `MaxBodyCells` cells along either axis, so a
  wall column is one entity and no body is ever large enough to confuse the wrapped images the
  space and the raycast work with. The collision solver then never leaves a unit inside a wall —
  `Static` means infinite mass — and the bodies are in the space, so vision sees them and they
  occlude. A game turns this on explicitly: `board.NewPlugin(...).WithCollision(c)`; the bodies
  are rebuilt whenever `TerrainMap.Version` moves, and once after a load, where the saved ones are
  replaced by what the terrain says. Their `Caps` are settled by collision on the next tick, so an
  edit to the terrain mid-game is solid one tick late.
- **A hole — open.** Not a body: a body would occlude sight, and a `Sensor` would fire the moment a
  unit touched its edge. A unit falls when its **centre** stands over a hole (its centre of mass
  is unsupported); the board reports `Fell{ID, Cell, Kind}` to `plugin.Each` behaviors, and the
  game decides: despawn, teleport, damage.
- **A forest — an interim step.** `CellKind.Opaque` makes a passable cell a body without a
  `Collider`: it occludes and nothing else. Every entry in the space occludes completely today, so
  a unit inside a forest sees nothing until §12 lands.
- **After a push.** The existing re-plan on being knocked off a `Leg` covers it, and covers being
  pushed onto a passable cell off the route as well.

## 7. Waypoints

A `MoveOrder` keeps a queue of up to `MaxWaypoints` goals behind its `Target`. A goal with more
behind it is passed by projection like a waypoint, the next becomes the `Target` and is aimed at in
the same tick, so the unit does not stop at the intermediate ones; it brakes only before the
last. Navigation's default bindings, in the sense of [views.md](views.md): right click →
`MoveTo{Cell}`, "Move here" (replaces the order and its queue); Shift + right click →
`AddWaypoint{Cell}`, "Add waypoint" (appends; an idle unit gets a fresh order). Today
`DefaultCommandEventHandler` does both through `MoveCommand{Cell, Append}`; the labelled binding
comes with the players layer. The route renderer draws the way to every queued goal, planning
each leg once and keeping it until the goals change.

## 8. Drawing the route is a choice of style

`PathRenderer` draws the route cell by cell today (`PathSprites`: one arrow per direction, a
`Dot`), and that stays the default, because it is cheap: one sprite per cell. How a route is
drawn becomes a style, as `vision.ConeStyle` and `selection.HighlightStyle` are: a
`navigation.RouteStyle` with two ready-made — `CellArrows`, today's, and `SmoothRoute`, the line
the unit will actually follow: arcs of radius `Speed / TurnRate` through the lookahead points,
the queued waypoints marked, the last segment to the goal's centre — plus a `RouteStyleFn` for
one's own, chosen at `WithRenderer`/`WithStyle`.

`SmoothRoute`'s cost is held down two ways: routes are drawn for selected units only, as today;
and the polyline is a function of the path and the motion profile, so it is computed when the path
is re-planned or the waypoint queue changes and kept beside the `Path` — a frame only draws it.
The lookahead point in navigation and the arc sampling in the style share one piece of code, so
what is drawn and what is driven cannot drift apart.

## 9. Turn-based movement — open, its own plan

A strictly turn-based game wants every unit to move at one tempo, not its own, and wants to say
who moves when. That is a layer above navigation: it sets the tempo for the duration of a move
(overriding the profile's speed while the move lasts) and issues `MoveTo` one unit at a time,
waiting for each arrival. Nothing in navigation needs to know; it is planned separately, after
the waypoint queue.

## 10. Arbitration — open

The planner and a reaction may want to steer the same unit in the same tick: a unit on its route
while `Flee` says "sideways". The rule to start with: the reactive request wins the tick, and the
planner, finding itself off the route, re-plans (that exists). The alternative — a `Steering`
that sums weighted requests, in the manner of classic steering behaviors — is more machinery,
worth it only once the simple rule fails somewhere real.

## 11. What changes in code, when it comes to that

Not a plan, a list. `navigationSystem` becomes a source of `Request` calls and no longer writes
`Vel`; navigated units must carry `Steering` (their kind gives it); `arrivalEpsilon` applies to
the goal only. `Steering` grows the motion profile and `SteeringSystem` writes `Vel.Value` every
tick. `MoveOrder` keeps a queue of goals; the bindings "Move here" and "Add waypoint" replace the
command event handler. `RouteStyle` with `CellArrows` (default) and `SmoothRoute` (opt-in, cached
per path change). The board gets terrain bodies (done: `Grid.CellBoxes`, `Body`, `WithCollision`) and, still to
come, `Fell` for holes; static entities take their cells in `Occupancy`.

Tests, when it comes to that: a smooth turn (the heading's angle changes monotonically and the
speed never drops to zero at a bend); acceleration from `V0` to `MaxSpeed` by `Accel`; braking that
ends on the goal's centre; a terrain modifier that does not compound across ticks; passing a
waypoint by projection; entering a cell independently of waypoints; a push into a wall leaves no
unit inside it (done, on a square and on a hex grid); a unit centred over a hole yields a `Fell`.

## 12. Sight through terrain, sight range and flying units — open

Terrain should limit sight by kind, not switch it off: a forest takes a few cells of range, a hill
none, a wall all. That needs three things, in this order.

- **Transparency per kind.** `CellKind.Transparency` in 0..1 replaces the binary `Opaque`: the
  range a ray has left shrinks by the cell's share as it crosses. The raycast in aabbworld knows
  only hit or miss, so `Space.Scan` grows an optional per-entry attenuation — a function of the
  entry's id, or a capability bit "translucent" plus the depth crossed — and vision maps a terrain
  body's `TypeID` (each kind of body has its own, from `Kinds.Reserve`) to the attenuation of its
  kind. Terrain bodies then get one `TypeID` per `CellKind`, not one for the whole board.
- **Range is the unit's.** `Sight.Radius` already is; `vision.MaxSightRadius` caps it at 300 for
  the outline buffers and moves into configuration or grows with the largest radius defined.
- **Flying units.** A `world.Flying` tag (or a `Base.Caps` bit): collision's `Touch` vetoes its pairs
  with terrain bodies the way it vetoes a lost `Collider`, vision does not attenuate its sight, and
  the planner ignores `Passable` and `Cost` for it — `pathFinder` takes a per-order filter
  (`MoveOrder.Flying`, or the tag read at planning). Cells taken by static entities (§5) still
  count for it. Order of work: 4b holes, 4c cells taken by static entities, 4d transparency and
  flying.

## Who owns what

| Concept | Owner | Promises |
|:---|:---|:---|
| turning, accelerating, the motion profile | `world.Steering`, `SteeringSystem` | "asked for a heading and a speed, the unit gets there as its profile allows, and `Vel` is rewritten every tick" |
| the route, the lookahead point, waypoints, when to brake | `navigation` | "the unit is asked, every tick, for the heading and speed that keep it on its route" |
| terrain, walls as bodies, who fell into a hole, cells taken by static entities | `board` | "what may not be entered is a body in the world, and the planner knows it; a hole says who fell in" |
| pushing apart, contacts, `Static` and `Sensor` | `collision` | "no unit ends a tick inside a wall" |
