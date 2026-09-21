// Package kind is how a game says what its entities are. A Spec lists the components a kind
// carries, each the same for all (Const) or read from the entity's own row (Load); Define
// registers it with a world, and the kind's Entry puts one entity on the roster.
package kind
