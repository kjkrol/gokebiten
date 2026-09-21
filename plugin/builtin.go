package plugin

// Builtin marks a Plugin the engine installs itself; Use rejects installing it directly.
type Builtin interface{ Builtin() }
