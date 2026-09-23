// Package plugin is the extension contract: what a [Plugin] is, what it is handed at install
// time, and how game logic is hosted inside a plugin's own pass as a behavior. Built-in plugins
// and third-party ones implement exactly the same interface.
//
// # Plugin
//
// A [Plugin] has a Name (saves match its state by it, and Use rejects a duplicate), an Install
// that queues its ECS wiring, a RunPlan the game calls once a tick in the order it needs, and
// optional faces: WithRenderer and Renderer for what it draws, EventHandler for the input it
// reads, Serializable for the state it saves, RegisterBehavior for the behaviors it hosts. A
// [Builtin] plugin is one the engine installs itself, such as the world; Use refuses it.
//
// # Installer
//
// [Installer] is ECS wiring and nothing else: UseModule, Setup, RegSys, ECS. Install only queues;
// the engine flushes every plugin's wiring in one ecs.Setup after the Stage's Init. Cross-plugin
// data comes from constructor injection, not from the Installer.
//
// # Behaviors
//
// A [Behavior] is game logic a plugin runs inside its own pass, built with [Between] — react to
// every pair the host meets where one entity carries tag a and the other b, [Any] standing for
// either side — or [Each] — react on every entity the host visits that carries T. The payload
// type P is what says which plugin hosts it; a host refuses another's with [ErrUnhostedBehavior],
// and one registered after the host's queries were built with [ErrHostBuilt]. Register before Use.
//
// A [Tag] is a bit of a family: [Tags] is the family's component, holding up to
// [MaxTagsPerFamily] of them, and an empty type of the plugin's or the game's names the family.
// The families a host's behaviors name join its queries as optional components, so a behavior
// costs no query of its own, and a host reads what an entity carries as [Marks] — what a payload
// passes on for [Carries]. One PairHost's behaviors may name at most [MaxFamilies] families.
//
// A plugin hosts behaviors with [PairHost] (Bind its families to the host's queries once, then
// Dispatch, DispatchEitherWay or DispatchGrouped per pair or per observer) and [EachHost] (Bind,
// then Run over each chunk walked).
//
// # Tick
//
// [Tick] is what a behavior is told about the pass it runs in: the command buffer its structural
// changes go through (they land when the pass is over), the time read once for the whole pass,
// and the tick's length.
//
// # Optional interfaces
//
// [Serializable] contributes pointers for Persistence to encode and decode. [Populator] seeds its
// own initial state, run only when a Stage starts without a restored save. [Restorer] is called
// right after its state was decoded; [PostLoader] supplies a one-time system run after the
// entities were restored, for a hook that must query them.
package plugin
