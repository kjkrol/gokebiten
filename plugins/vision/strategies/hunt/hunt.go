// Package hunt is a ready-made reaction to what an entity sees: it goes after
// the nearest of what it was shown, and with none in view it searches — a
// quarter turn to one side or the other, a stretch straight ahead, and again.
//
// Hand Chase to plugin.Between — Between[hunt.Predator, hunt.Prey], or any two
// tags of the game's own — and register that with vision.Plugin.RegisterBehavior.
// Who hunts whom is the registration's to say; what a catch means is the game's
// own business — this package only steers.
package hunt

import (
	"math/rand/v2"
	"time"

	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// Predator and Prey are ready-made tags for the two sides of a hunt — any two tags of the game's own do as well.
type Predator struct{}

// Prey is what a Predator is shown; anything else it sees is scenery.
type Prey struct{}

// Chase steers the observer at the nearest entity it was shown. Shown none, it
// turns a quarter to a random side every lookEvery, and runs straight in
// between — never, for a lookEvery of zero.
func Chase(lookEvery time.Duration) func(plugin.Tick, vision.Sighting) {
	started := time.Now()
	return func(t plugin.Tick, s vision.Sighting) {
		if s.Steering == nil {
			return
		}
		if len(s.Seen) > 0 {
			chase(s)
			return
		}
		if lookEvery > 0 && looksDue(started, t, lookEvery) {
			lookAside(s)
		}
	}
}

// chase points the predator at the nearest prey — Seen comes nearest first.
func chase(s vision.Sighting) {
	ox, oy := centre(&s.Base.Pos)
	tx, ty := centre(&s.Seen[0].Base.Pos)
	if tx != ox || ty != oy {
		s.Steering.Request(geom.NewVec(tx-ox, ty-oy))
	}
}

// looksDue reports the tick in which another stretch of every has run out.
func looksDue(started time.Time, t plugin.Tick, every time.Duration) bool {
	now := t.Now.Sub(started)
	return now/every != (now-t.Dt)/every
}

// lookAside turns the predator a quarter to whichever side the coin falls.
func lookAside(s vision.Sighting) {
	heading := s.Base.Vel.Dir
	if heading.X == 0 && heading.Y == 0 {
		return
	}
	side := geom.NewVec(-heading.Y, heading.X)
	if rand.IntN(2) == 0 {
		side = geom.NewVec(heading.Y, -heading.X)
	}
	s.Steering.Request(side)
}

func centre(p *world.Position) (float64, float64) {
	box := p.AABB.AABB
	return (float64(box.TopLeft.X) + float64(box.BottomRight.X)) / 2,
		(float64(box.TopLeft.Y) + float64(box.BottomRight.Y)) / 2
}
