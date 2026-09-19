// Package flee is a ready-made world.Behavior: an entity marked Skittish steers
// away from everything its Sight picked up, the nearest threat pulling hardest.
//
// Register it with world.Plugin.RegisterBehavior, then give Skittish to
// whichever kinds should run from what they see.
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

// Skittish marks the entities this behavior steers. Nothing else carries it, so
// the query visits only the chunks those entities live in.
type Skittish struct{}

var _ world.Behavior = (*Behavior)(nil)

// Behavior steers Skittish entities away from what they see.
type Behavior struct {
	off bool // zero value is on

	query   *goke.Query
	sighted goke.Comp[vision.Sighted]
	steer   goke.Comp[world.Steering]
	pos     goke.Comp[world.Position]

	// lookup resolves a sighted id back to its Position — the scan reports who
	// and how far, not where.
	lookup    *goke.Query
	seenPos   goke.Comp[world.Position]
	lookupHot bool
}

func New() *Behavior { return &Behavior{} }

// SetEnabled turns the behavior off and on again in place, without
// unregistering it — useful for showing what the avoidance is actually worth,
// and for game states where an entity should stop caring what it sees.
func (b *Behavior) SetEnabled(on bool) { b.off = !on }

func (b *Behavior) Init(si *goke.SysInit) {
	si.RegComp[Skittish]()
	b.query = si.NewQueryBuilder(&b.sighted, &b.steer, &b.pos).
		Include(goke.Include[Skittish]()).
		Build()
	b.lookup = si.NewQueryBuilder(&b.seenPos).Build()
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

		for i := range cursor.IDs {
			if seen[i].Count == 0 {
				continue
			}
			if away, ok := b.awayFrom(&positions[i], &seen[i]); ok {
				// Refused while the entity is still reacting to an earlier
				// decision — that refusal is the point of Reflex.
				steers[i].Request(away)
			}
		}
	}
}

// awayFrom sums one push per sighting, weighted by 1/distance so the nearest
// dominates and something far off barely bends the course. Steering.Request
// normalises whatever comes out.
func (b *Behavior) awayFrom(from *world.Position, seen *vision.Sighted) (geom.Vec, bool) {
	ox, oy := centre(from)

	var sum geom.Vec
	for i := range int(seen.Count) {
		other, ok := b.positionOf(seen.IDs[i])
		if !ok {
			continue // gone since the scan
		}
		tx, ty := centre(other)
		dx, dy := ox-tx, oy-ty
		d := math.Hypot(dx, dy)
		if d == 0 {
			continue
		}
		sum.X += dx / (d * d)
		sum.Y += dy / (d * d)
	}
	return sum, sum.X != 0 || sum.Y != 0
}

func (b *Behavior) positionOf(id uid.UID64) (*world.Position, bool) {
	ok := b.lookupHot && b.lookup.SeekH(id)
	if !ok {
		ok = b.lookup.Seek(id)
		b.lookupHot = ok
	}
	if !ok {
		return nil, false
	}
	return b.seenPos.At(b.lookup.Cursor()), true
}

func centre(p *world.Position) (float64, float64) {
	box := p.AABB.AABB
	return (float64(box.TopLeft.X) + float64(box.BottomRight.X)) / 2,
		(float64(box.TopLeft.Y) + float64(box.BottomRight.Y)) / 2
}
