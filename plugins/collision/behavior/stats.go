package behavior

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collision"
)

// CountContacts adds every contact it is handed to a ContactStats the game owns.
func CountContacts(stats *ContactStats) func(plugin.Tick, collision.Meeting) {
	return func(plugin.Tick, collision.Meeting) { stats.Counter++ }
}

// ContactStats is the running total of contacts CountContacts maintains. It only grows:
// whoever shows a rate works it out from the total, as render.TelemetryRenderer does.
type ContactStats struct {
	Counter int
}

// Reset zeroes Counter, for a game that wants to start counting afresh.
func (s *ContactStats) Reset() { s.Counter = 0 }
