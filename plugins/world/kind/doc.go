// Package kind is how a game says what its entities are. A Spec lists the components a kind
// carries, each the same for all (Const) or read from the entity's own row (Load); Define
// registers it with a world, and the kind's Entry puts one entity on the roster.
//
// # Spec, Const and Load
//
// A [Spec] is the list of a kind's components, each a [Comp] made by [Const] (one value for every
// entity of the kind) or [Load] (a value read from each entity's own row of type P). Both return a
// [Template], whose WithEffect runs a callback right after the component is written for each
// spawned entity. A world.Position and a world.Velocity must each appear once, or Define panics
// by name; so does a Load over a row type other than the kind's.
//
// # Define and Of
//
// [Define] registers a Spec under a name with a [Registry] — a world's, reached through
// world.Plugin.Kinds — and hands back the kind as [Of], typed by its row: Entry builds one
// [Entry] for world.Plugin.Seed with the row checked by the compiler, ID is the [ID] every entity
// of the kind carries, SpriteID is the atlas slot its entities are drawn from, Name is what it
// was defined as.
//
// # Registry
//
// [Registry] is where kinds are kept; kind never imports world (world imports kind), which is
// why a kind's id is [ID] and the registry sits behind this interface. It also lets saves know
// every component type a kind gives its entities. One Registry holds at most [MaxKinds] kinds.
package kind
