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
a `Request`. An AI may speak at either level.

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

Everything below follows from the profile. The turning radius is `Speed / TurnRate`: wide arcs at
full speed, tight ones when setting off. The braking distance is `Speed² / (2·Accel)`: navigation
asks for `WantSpeed = 0` that far from the goal and the unit comes to rest on it. The profile is
the kind's (`kind.Const(world.Steering{MaxSpeed: 120, Accel: 200, V0: 40, TurnRate: 0.1})`), so
unit types differ in how they move without any code.

## 2. Where to look: the lookahead point

Instead of the centre of the current cell, the unit aims at a point on its path a distance
`d = Speed / TurnRate` ahead — the arc it can physically make. Walk the path's segments from the
unit's projection onto them, lay off `d`, take the point (on a torus through `shortestAxisDelta`,
as today). As the point passes a bend in the path, the requested heading starts to rotate before
the unit reaches the cell, and the `SteeringSystem` carries it round the arc. With `TurnRate = 0`
the point is the next waypoint and the unit moves as it does today.

Speed stays constant along the route. Optionally it scales with `max(cos θ, floor)` of the angle
between `Vel.Dir` and `Want`, so a sharp bend slows the unit rather than stopping it. Slowing to a
halt happens only at the end: `WantSpeed = 0` from the braking distance, then the nudge onto the
goal's centre as today.

## 3. Passing a waypoint is not being near its centre

A waypoint is passed when the unit's projection onto the waypoint's segment goes beyond the
segment's length — it has crossed the plane perpendicular to the segment at the waypoint. It
never has to touch the centre. `arrivalEpsilon` remains for the final goal only.

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

What happens when a collision pushes a unit onto a cell it may not enter is the cell kind's
decision (`board.CellKind`), and both answers already have their machinery:

- **A wall.** Impassable cells become **wall entities** spawned by the board — a kind "wall" with
  `Position`, `collision.Collider` and `collision.Physics{Mass: +Inf}` — one per run of adjacent
  cells in a row, rebuilt when the terrain changes. The solver then never leaves a unit inside a
  wall; "infinite mass" is what `Static` means; and walls are in saves, in every `View` and in
  vision (they occlude) for free. The cost is entities in the Space: runs, not cells.
- **A hole.** A cell entity with a `Collider` and no `Physics` — a `Sensor`. Contact is a
  `collision.Meeting` carrying a tag (`board.Hole`), and a behavior decides: despawn, teleport,
  damage. This is today's reaction machinery, nothing new.
- **After a push.** The existing re-plan on being knocked off a `Leg` covers it, and covers being
  pushed onto a passable cell off the route as well.

## 7. Waypoints

A `MoveOrder` keeps a queue of goals, not one `Target`. The path is planned to the nearest, and
the lookahead point runs through a waypoint into the next segment, so the unit does not stop at
the intermediate ones; it brakes only before the last. Navigation's default bindings, in the
sense of [views.md](views.md): right click → `MoveTo{Cell}`, "Move here" (replaces the queue);
Shift + right click → `AddWaypoint{Cell}`, "Add waypoint" (appends). Today's
`DefaultCommandEventHandler` is the first of these.

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

## 9. Arbitration — open

The planner and a reaction may want to steer the same unit in the same tick: a unit on its route
while `Flee` says "sideways". The rule to start with: the reactive request wins the tick, and the
planner, finding itself off the route, re-plans (that exists). The alternative — a `Steering`
that sums weighted requests, in the manner of classic steering behaviors — is more machinery,
worth it only once the simple rule fails somewhere real.

## 10. What changes in code, when it comes to that

Not a plan, a list. `navigationSystem` becomes a source of `Request` calls and no longer writes
`Vel`; navigated units must carry `Steering` (their kind gives it); `arrivalEpsilon` applies to
the goal only. `Steering` grows the motion profile and `SteeringSystem` writes `Vel.Value` every
tick. `MoveOrder` keeps a queue of goals; the bindings "Move here" and "Add waypoint" replace the
command event handler. `RouteStyle` with `CellArrows` (default) and `SmoothRoute` (opt-in, cached
per path change). The board gets the kinds "wall" and "hole" and a system rebuilding them when
terrain changes; static entities take their cells in `Occupancy`.

Tests, when it comes to that: a smooth turn (the heading's angle changes monotonically and the
speed never drops to zero at a bend); acceleration from `V0` to `MaxSpeed` by `Accel`; braking that
ends on the goal's centre; a terrain modifier that does not compound across ticks; passing a
waypoint by projection; entering a cell independently of waypoints; a push into a wall leaves no
unit inside it; a push into a hole yields a `Meeting`.

## Who owns what

| Concept | Owner | Promises |
|:---|:---|:---|
| turning, accelerating, the motion profile | `world.Steering`, `SteeringSystem` | "asked for a heading and a speed, the unit gets there as its profile allows, and `Vel` is rewritten every tick" |
| the route, the lookahead point, waypoints, when to brake | `navigation` | "the unit is asked, every tick, for the heading and speed that keep it on its route" |
| terrain, walls and holes as entities, cells taken by static entities | `board` | "what may not be entered is a body or a trap in the world, and the planner knows it" |
| pushing apart, contacts, `Static` and `Sensor` | `collision` | "no unit ends a tick inside a wall; a hole says who fell in" |
