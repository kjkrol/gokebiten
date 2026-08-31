package board

// MaxPathLength bounds Path.Steps — a route longer than this is fetched in successive chunks.
const MaxPathLength = 64

// Path is the cached route toward MoveTo.Target, consumed step by step by
// SteeringSystem — Length 0 means "not computed yet".
type Path struct {
	Steps  [MaxPathLength]CellID
	Length uint16
	Index  uint16
}
