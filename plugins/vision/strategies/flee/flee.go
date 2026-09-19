// Package flee is a ready-made world.Behavior: an entity marked Skittish gives
// way to whatever its Sight says is on a collision course with it.
//
// Register it with world.Plugin.RegisterBehavior, then give Skittish to
// whichever kinds should dodge. What it passes by is left alone — only two
// things earn a swerve: something heading at the entity, and the entity
// heading at something.
package flee

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// onCourse is how straight one has to be going at the other to count, as the
// cosine of the angle between them — anything wider than this passes by, and
// a demand for dead-on (zero) would make every crossing path a threat.
const onCourse = 0.5

// Skittish marks the entities this behavior steers. Nothing else carries it, so
// the query visits only the chunks those entities live in.
type Skittish struct{}

var _ world.Behavior = (*Behavior)(nil)

// Behavior steers Skittish entities away from what is closing on them.
type Behavior struct {
	off bool // zero value is on

	query   *goke.Query
	sighted goke.Comp[vision.Sighted]
	steer   goke.Comp[world.Steering]
	pos     goke.Comp[world.Position]
	vel     goke.Comp[world.Velocity]

	// lookup resolves a sighted id back to where it is and where it is going —
	// the scan reports who and how far, not that.
	lookup    *goke.Query
	seenPos   goke.Comp[world.Position]
	seenVel   goke.OptComp[world.Velocity]
	lookupHot bool
}

func New() *Behavior { return &Behavior{} }

// SetEnabled turns the behavior off and on again in place, without
// unregistering it — useful for showing what the avoidance is actually worth,
// and for game states where an entity should stop caring what it sees.
func (b *Behavior) SetEnabled(on bool) { b.off = !on }

func (b *Behavior) Init(si *goke.SysInit) {
	si.RegComp[Skittish]()
	b.query = si.NewQueryBuilder(&b.sighted, &b.steer, &b.pos, &b.vel).
		Include(goke.Include[Skittish]()).
		Build()
	b.lookup = si.NewQueryBuilder(&b.seenPos).Optional(&b.seenVel).Build()
}

func (b *Behavior) Update(*goke.CmdBuf, time.Duration) {
	if b.off {
		return
	}
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		seen := b.sighted.Slice(cursor)
		steers := b.steer.Slice(cursor)
		positions := b.pos.Slice(cursor)
		velocities := b.vel.Slice(cursor)

		for i := range cursor.IDs {
			if seen[i].Count == 0 {
				continue
			}
			if away, ok := b.awayFrom(&positions[i], velocities[i].Dir, &seen[i]); ok {
				// Refused while the entity is still reacting to an earlier
				// decision — that refusal is the point of Reflex.
				steers[i].Request(away)
			}
		}
	}
}

// awayFrom sums one push per sighting that is on a collision course, weighted
// by 1/distance so the nearest dominates and something far off barely bends the
// course. Steering.Request normalises whatever comes out.
func (b *Behavior) awayFrom(from *world.Position, heading geom.Vec, seen *vision.Sighted) (geom.Vec, bool) {
	ox, oy := centre(from)

	var sum geom.Vec
	for i := range int(seen.Count) {
		other, otherVel, ok := b.seenAt(seen.IDs[i])
		if !ok {
			continue // gone since the scan
		}
		tx, ty := centre(other)
		dx, dy := ox-tx, oy-ty
		d := math.Hypot(dx, dy)
		if d == 0 {
			continue
		}
		if !closing(heading, otherVel, -dx/d, -dy/d) {
			continue
		}
		sum.X += dx / (d * d)
		sum.Y += dy / (d * d)
	}
	return sum, sum.X != 0 || sum.Y != 0
}

// closing reports whether the two are set to meet: either this entity is
// heading at the other, or the other is heading at it. towardsX/Y is the unit
// direction from here to there.
func closing(heading geom.Vec, otherVel *world.Velocity, towardsX, towardsY float64) bool {
	if heading.X*towardsX+heading.Y*towardsY > onCourse {
		return true
	}
	if otherVel == nil {
		return false
	}
	return otherVel.Dir.X*towardsX+otherVel.Dir.Y*towardsY < -onCourse
}

func (b *Behavior) seenAt(id uid.UID64) (*world.Position, *world.Velocity, bool) {
	ok := b.lookupHot && b.lookup.SeekH(id)
	if !ok {
		ok = b.lookup.Seek(id)
		b.lookupHot = ok
	}
	if !ok {
		return nil, nil, false
	}
	cursor := b.lookup.Cursor()
	return b.seenPos.At(cursor), b.seenVel.At(cursor), true
}

func centre(p *world.Position) (float64, float64) {
	box := p.AABB.AABB
	return (float64(box.TopLeft.X) + float64(box.BottomRight.X)) / 2,
		(float64(box.TopLeft.Y) + float64(box.BottomRight.Y)) / 2
}
