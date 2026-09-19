package world

import (
	"math"

	"github.com/kjkrol/gokg/geom"
)

// Steering turns a requested heading into motion gradually instead of at once:
// Reflex ticks pass before a request takes effect, and TurnRate caps how fast
// Velocity.Dir may swing towards it afterwards.
//
// The two are different things. Reflex is how long the entity takes to react —
// and, while it counts down, the entity cannot change its mind, so something
// that has just committed to a turn keeps turning when a newer threat appears.
// TurnRate is how sharply it can turn once it does. An entity with neither set
// follows every request immediately and exactly.
//
// Add it to whatever should not react instantly, whether it is driven by a
// Behavior, by a path, or by the player.
type Steering struct {
	Want     geom.Vec // heading asked for; zero means nothing is standing
	TurnRate float64  // radians per tick, 0 to swing all the way at once
	Reflex   uint8    // ticks between a request and acting on it
	Delay    uint8    // ticks still to wait
}

// Request asks the entity to head towards dir, which need not be a unit vector
// — a behavior may hand over the sum of whatever is pushing it around. It is
// refused while an earlier request is still being reacted to, and reports
// whether it was taken.
func (s *Steering) Request(dir geom.Vec) bool {
	if s.Delay > 0 {
		return false
	}
	if n := math.Hypot(dir.X, dir.Y); n > 0 {
		dir = geom.NewVec(dir.X/n, dir.Y/n)
	}
	s.Want = dir
	s.Delay = s.Reflex
	return true
}
