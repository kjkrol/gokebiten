package stats

// Stats is the aggregate collision-count telemetry this strategy maintains — see physics.Plugin.EnableStats.
type Stats struct {
	Counter int
}

// Reset zeroes Counter — called each stats interval when Stats is a registered Resource.
func (s *Stats) Reset() { s.Counter = 0 }
