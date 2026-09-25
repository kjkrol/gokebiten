package comp

import "github.com/kjkrol/gram/plugin"

// Tagged is a Spec component giving every entity of the kind these tags of one family; a Spec
// names each family once. The tags come from world.Kinds.DefineTag.
func Tagged[F any](tags ...plugin.Tag[F]) Template[plugin.Tags[F]] {
	return Const(plugin.Tags[F](0).With(tags...))
}
