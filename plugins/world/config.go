package world

import (
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/gram/camera"
)

// Config configures world's spatial shape and the bounds its entity population must respect.
type Config struct {
	Space    SpaceCfg
	Entities EntitiesCfg
	Camera   camera.Config
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
