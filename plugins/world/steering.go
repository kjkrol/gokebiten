package world

import (
	"math"

	"github.com/kjkrol/aabbworld/geom"
)

// Steering turns a requested heading into motion gradually: Reflex ticks pass before a request
// takes effect, and TurnRate caps how fast Velocity.Dir swings towards it afterwards.
type Steering struct {
	Want     geom.Vec // heading being turned towards; zero means none yet
	Pending  geom.Vec // heading asked for, taking over from Want once Delay runs out
	TurnRate float64  // radians per tick, 0 to swing all the way at once
	Reflex   uint8    // ticks between a request and acting on it
	Delay    uint8    // ticks still to wait
}

// Request asks the entity to head towards dir, any length; false while an earlier one is pending.
func (s *Steering) Request(dir geom.Vec) bool {
	if s.Delay > 0 {
		return false
	}
	if n := math.Hypot(dir.X, dir.Y); n > 0 {
		dir = geom.NewVec(dir.X/n, dir.Y/n)
	}
	if s.Reflex == 0 {
		s.Want = dir
		return true
	}
	s.Pending = dir
	s.Delay = s.Reflex
	return true
}
