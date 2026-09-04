package resource

// PluginResource marks a type as valid data for plugins.GameCtx.Provide/Require
// and plugins.Resources — plain information a plugin publishes for others to
// read, never a Plugin itself or other behavior-bearing type.
type PluginResource interface {
	PluginResource()
}
