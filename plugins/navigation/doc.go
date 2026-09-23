// Package navigation moves entities along a MoveOrder's path across a
// board, re-pathing automatically when terrain along the route changes.
// WithCommands adds right-click move orders for Selected entities;
// WithRenderer draws the remaining route.
//
// # MoveOrder and Path
//
// A [MoveOrder] on an entity commands it toward a Target cell until it arrives, when the order is
// removed. Its [Path] is the cached route, consumed step by step, at most [MaxPathLength] cells at
// a time with a longer route fetched in chunks; its [Leg] is the single step in flight — every
// cell it holds in Occupancy until it reaches the next centre. [CellEntered] is a one-tick tag
// added the tick an entity's Cell changes. A navigated entity carries a world.Steering profile:
// navigation only asks it for a heading at the lookahead point and for its own top speed, braking
// from the profile before the goal. The [Plugin], built over a board and a world, runs before the
// world's RunPlan.
//
// # Commands
//
// WithCommands turns a right-click into a move order for every Selected entity: the
// [DefaultCommandEventHandler] writes the target into [Resources], and the command system issues
// the orders.
//
// # Renderer
//
// [Plugin.WithRenderer] draws the remaining route of every selected entity with the [PathRenderer]
// from a [PathSprites] set — one arrow per [Direction] and a dot; [RegisterDefaultPathSprites]
// bakes a default set.
package navigation
