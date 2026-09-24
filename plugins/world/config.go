package world

import (
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/gram/camera"
)

// Config configures world's spatial shape, the bounds its entity population must respect and
// whether it has heights.
type Config struct {
	Space    SpaceCfg
	Entities EntitiesCfg
	Camera   camera.Config
	// Quasi3D gives the world heights: entities carry a Z, terrain an altitude, sight an eye. A flat
	// world (the default) is a set of planes — see Layers — and refuses heights where it meets them.
	Quasi3D bool
}

type SpaceCfg struct {
	Width, Height uint32
	Edges         aabbworld.Edges
}

// EntitiesCfg bounds how many entities the world holds and the sizes they may spawn with.
type EntitiesCfg struct {
	MaxCount int
	MinSize  uint32
	MaxSize  uint32
}

type Telemetry struct {
	Count int
}
