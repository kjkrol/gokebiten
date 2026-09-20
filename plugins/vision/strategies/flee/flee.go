// Package flee is a ready-made reaction to what an entity sees: it gives way to
// whatever is on a collision course with it.
//
// Hand Behavior.Steer to plugin.Between — Between[flee.Skittish,
// plugin.Anything], with plugin.Asking[flee.Threat]() — and register that with
// vision.Plugin.RegisterBehavior. What the entity passes by is left alone: only
// two things earn a swerve, something heading at it, and it heading at
// something. A Threat is the exception: it is run from on sight, and while one
// is in view nothing else gets a say in where to run.
package flee

import (
	"math"

	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// onCourse is how straight one has to be going at the other to count, as the
// cosine of the angle between them — anything wider than this passes by, and
// a demand for dead-on (zero) would make every crossing path a threat.
const onCourse = 0.5

// Skittish is a ready-made tag for the entities to steer — any tag of the game's own does as well.
type Skittish struct{}

// Threat marks an entity that is fled from the moment it comes into view,
// course or no course — what a predator is given so nothing waits to be aimed at.
type Threat struct{}

// Behavior steers Skittish entities away from what is closing on them.
type Behavior struct {
	off bool // zero value is on
}

func New() *Behavior { return &Behavior{} }

// SetEnabled turns the behavior off and on again in place, without
// unregistering it — useful for showing what the avoidance is actually worth,
// and for game states where an entity should stop caring what it sees.
func (b *Behavior) SetEnabled(on bool) { b.off = !on }

// Steer turns the observer away from what it sees closing on it. It asks its
// Seen about Threat, so the Between it is handed to has to declare plugin.Asking[Threat]().
func (b *Behavior) Steer(_ plugin.Tick, s vision.Sighting) {
	if b.off || s.Steering == nil {
		return
	}
	if away, ok := awayFrom(s); ok {
		// Refused while the entity is still reacting to an earlier decision —
		// that refusal is the point of Reflex.
		s.Steering.Request(away)
	}
}

// awayFrom sums one push per sighting worth avoiding, weighted by 1/distance so
// the nearest dominates and something far off barely bends the course. Threats
// are summed apart from everything else on a collision course, and win outright:
// flight is straight away from them, not a compromise with a neighbour in the
// way. Steering.Request normalises whatever comes out.
func awayFrom(s vision.Sighting) (geom.Vec, bool) {
	ox, oy := centre(&s.Base.Pos)
	heading := s.Base.Vel.Dir

	var threats, others geom.Vec
	for _, seen := range s.Seen {
		tx, ty := centre(&seen.Base.Pos)
		dx, dy := ox-tx, oy-ty
		d := math.Hypot(dx, dy)
		if d == 0 {
			continue
		}
		switch {
		case seen.Carries[Threat]():
			threats.X += dx / (d * d)
			threats.Y += dy / (d * d)
		case closing(heading, seen.Base.Vel.Dir, -dx/d, -dy/d):
			others.X += dx / (d * d)
			others.Y += dy / (d * d)
		}
	}
	if threats.X != 0 || threats.Y != 0 {
		return threats, true
	}
	return others, others.X != 0 || others.Y != 0
}

// closing reports whether the two are set to meet: either this entity is
// heading at the other, or the other is heading at it. towardsX/Y is the unit
// direction from here to there.
func closing(heading, otherHeading geom.Vec, towardsX, towardsY float64) bool {
	if heading.X*towardsX+heading.Y*towardsY > onCourse {
		return true
	}
	return otherHeading.X*towardsX+otherHeading.Y*towardsY < -onCourse
}

func centre(p *world.Position) (float64, float64) {
	box := p.AABB.AABB
	return (float64(box.TopLeft.X) + float64(box.BottomRight.X)) / 2,
		(float64(box.TopLeft.Y) + float64(box.BottomRight.Y)) / 2
}
