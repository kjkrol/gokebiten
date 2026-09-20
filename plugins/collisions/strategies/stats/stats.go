// Package stats is a ready-made reaction to contacts: a running count of how
// many happened, for telemetry.
//
// Hand Count to plugin.Between and register that with
// collisions.Plugin.RegisterBehavior — Between[plugin.Anything, plugin.Anything]
// to count every confirmed contact, detected-only ones included, or a pair of
// tags to count just those. It rides the narrow phase's own pass, at the cost
// of no query of its own.
package stats

import (
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
)

// Count adds every contact it is handed to a Stats counter the game owns.
func Count(stats *Stats) func(plugin.Tick, collisions.Meeting) {
	return func(plugin.Tick, collisions.Meeting) { stats.Counter++ }
}
