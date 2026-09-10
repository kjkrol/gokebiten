package world

// Config configures world's spatial shape and the bounds its entity population must respect.
type Config struct {
	Space    SpaceCfg
	Entities EntitiesCfg
}

type SpaceCfg struct {
	Width, Height uint32
	Toroidal      bool
}

type EntitiesCfg struct {
	MaxCount int
	MinSize  uint32
	MaxSize  uint32
}

type Telemetry struct {
	Count int
}
