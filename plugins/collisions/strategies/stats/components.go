package stats

// Stats is the running collision count this strategy maintains, for a game to read and display.
type Stats struct {
	Counter int
}

// Reset zeroes Counter, for a game showing a per-interval rate rather than a total.
func (s *Stats) Reset() { s.Counter = 0 }
