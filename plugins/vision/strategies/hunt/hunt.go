// Package hunt is a ready-made world.Behavior: a Predator goes after the
// nearest Prey its Sight picked up, and looks elsewhere once that one is gone.
//
// Register it with world.Plugin.RegisterBehavior, then give Predator to
// whichever kinds hunt and Prey to whichever are hunted. What a catch means is
// the game's own business — this package only steers.
package hunt

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

// Predator marks the entities this behavior steers, Prey what they go after.
type Predator struct{}

// Prey marks what a Predator chases; anything else it sees is scenery.
type Prey struct{}

var _ world.Behavior = (*Behavior)(nil)

// Behavior steers each Predator at the nearest Prey it can see.
type Behavior struct {
	query   *goke.Query
	sighted goke.Comp[vision.Sighted]
	steer   goke.Comp[world.Steering]
	pos     goke.Comp[world.Position]

	// lookup resolves a sighted id back to where it is, and says whether it is
	// prey at all — the scan reports who and how far, not what.
	lookup    *goke.Query
	seenPos   goke.Comp[world.Position]
	seenPrey  goke.OptComp[Prey]
	lookupHot bool
}

func New() *Behavior { return &Behavior{} }

func (b *Behavior) Init(si *goke.SysInit) {
	si.RegComp[Predator]()
	si.RegComp[Prey]()
	b.query = si.NewQueryBuilder(&b.sighted, &b.steer, &b.pos).
		Include(goke.Include[Predator]()).
		Build()
	b.lookup = si.NewQueryBuilder(&b.seenPos).Optional(&b.seenPrey).Build()
}

func (b *Behavior) Update(*goke.CmdBuf, time.Duration) {
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		seen := b.sighted.Slice(cursor)
		steers := b.steer.Slice(cursor)
		positions := b.pos.Slice(cursor)

		for i := range cursor.IDs {
			if chase, ok := b.towardsPrey(&positions[i], &seen[i]); ok {
				steers[i].Request(chase)
			}
		}
	}
}

// towardsPrey points at the nearest prey in sight — Sighted is ordered by
// distance, so the first one that turns out to be prey is the one to chase.
func (b *Behavior) towardsPrey(from *world.Position, seen *vision.Sighted) (geom.Vec, bool) {
	ox, oy := centre(from)
	for i := range int(seen.Count) {
		prey, ok := b.preyAt(seen.IDs[i])
		if !ok {
			continue
		}
		tx, ty := centre(prey)
		if tx == ox && ty == oy {
			continue
		}
		return geom.NewVec(tx-ox, ty-oy), true
	}
	return geom.Vec{}, false
}

// preyAt is the sighted entity's position, and false for anything that is not
// prey — Query.Seek finds any entity, so Prey has to be read, not assumed.
func (b *Behavior) preyAt(id uid.UID64) (*world.Position, bool) {
	ok := b.lookupHot && b.lookup.SeekH(id)
	if !ok {
		ok = b.lookup.Seek(id)
		b.lookupHot = ok
	}
	if !ok {
		return nil, false
	}
	cursor := b.lookup.Cursor()
	if b.seenPrey.At(cursor) == nil {
		return nil, false
	}
	return b.seenPos.At(cursor), true
}

func centre(p *world.Position) (float64, float64) {
	box := p.AABB.AABB
	return (float64(box.TopLeft.X) + float64(box.BottomRight.X)) / 2,
		(float64(box.TopLeft.Y) + float64(box.BottomRight.Y)) / 2
}
