package world

import "github.com/kjkrol/gokg/plane"

// Position is an entity's world-space rectangle — shared by every plugin
// that places or draws entities (physics, board, ...).
type Position struct {
	plane.AABB[uint32]
}
