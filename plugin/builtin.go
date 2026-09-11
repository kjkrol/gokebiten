package plugin

// Builtin marks a Plugin as installed automatically by the engine — Use
// rejects any attempt to install it directly; see plugins/world.Plugin
// for the reference implementation.
type Builtin interface{ Builtin() }
