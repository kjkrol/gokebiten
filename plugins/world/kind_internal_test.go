package world

import "github.com/kjkrol/gram/plugins/world/kind"

// testKind is a kind as the registry would hold it, built straight from its parts.
func testKind(pos Position, vel Velocity, comps ...kind.Comp) registered {
	return registered{name: "test", position: kind.Const(pos), velocity: kind.Const(vel), comps: comps}
}
