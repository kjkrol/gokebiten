// Package world is the foundation for a Stage with moving, drawable entities: a Position and
// Velocity each, a shared spatial index other plugins query, and motion integrated each tick,
// never further than Position.MaxStep. What an entity carries is set by its kind — see kind.
//
// # Plugin
//
// [Plugin] is installed by a Stage's Init through ctx.UseWorld, once, from a [Config]: the
// [SpaceCfg] sizes the world and sets the edge rule per axis (aabbworld.Torus, WrapX or WrapY
// alone, OpenX or OpenY, a closed axis by default — a box stops whole at a closed edge, wraps at
// a wrapping one, may leave by an open one); the [EntitiesCfg] bounds how many entities the world
// holds and the sizes they spawn with; the camera.Config sizes the camera. An entity wholly past
// an open edge is handed, once, to [Plugin.OnExit] — despawned when no handler is set. The Plugin
// exposes the shared [aabbworld.Space] ([Plugin.Space]), the shared camera ([Plugin.Camera], with
// [Plugin.WithCameraControls] for wheel zoom, middle-drag pan and edge scroll), and
// [Plugin.Tracked] for a sibling plugin that moves boxes itself to report what the space said.
//
// # Base, Position and Velocity
//
// [Base] is the one component every entity carries: its [Position] (a plane.AABB), its
// [Velocity] (a unit heading and a speed in world units a second), the [kind.ID] it was spawned
// from, and the aabbworld.Capability bits the space indexes it under, which collision writes.
// A host hands Base to whatever it hosts instead of anyone binding it twice. No entity moves
// further in a tick than [StepReach] of its own shorter side ([Position.MaxStep],
// [Position.MaxSpeed]), so mixed sizes share a world without the smallest slowing the rest.
//
// # Kinds, Seed and Populate
//
// [Plugin.Kinds] is the registry kind.Define registers with; [Kinds] also issues atlas slots no
// kind owns ([Kinds.NewSprite]) and tells saves every component type its kinds carry. A Stage's
// Spawn puts entities on the roster with [Plugin.Seed]; the engine calls [Plugin.Populate] only
// when nothing was restored. [GridPlacement] arranges a population on a regular grid.
//
// # Attach, Detach and Declare
//
// [Plugin.Attach] and [Plugin.Detach] are the mid-game counterparts of a kind's Const, for game
// logic that has a plugin.Tick and no component id; [Plugin.Despawn] removes an entity at the end
// of the tick. [Plugin.Declare] tells saves about a type only ever attached; call it in Init.
//
// # Systems
//
// [Plugin.RunPlan] runs the tick: every registered [Behavior] (a decision system, see
// [Plugin.RegisterBehavior]), then [SteeringSystem] and [VelocitySystem] fold [Steering] requests
// and every [SpeedModifier] into each entity's speed, then [MoveSystem] moves every box under the
// edge rules and hands the space every Base as an aabbworld.Item — Space.Rebuild. The space keeps
// no state of its own between ticks: Populate and PostLoad rebuild it too, so it is whole before
// the first tick, and a despawned entity is gone from it on the next. Anything reading the space
// in its own pass sees the boxes as they stand after the last rebuild — and, after a collision
// tick, as the engine pushed them.
//
// # Appearance and Renderer
//
// [Appearance] is the sprite an entity is drawn from; [Plugin.WithRenderer] builds the entity
// [Renderer] over an atlas, and each [AppearanceModifier] it is given ([Renderer.WithModifier],
// WithOverlay, WithReplace, WithModify, [Facing]) resolves an entity's final draw layers in order.
// An [AppearanceStrategy] folds one override component into those layers.
//
// # Telemetry
//
// [Telemetry] counts the population; Resources publishes it for a renderer to show.
package world
