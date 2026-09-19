// Package elastic is a ready-made world.Behavior: an entity marked Bouncy
// takes its share of the impulse from everything it struck this tick.
//
// Register it with world.Plugin.RegisterBehavior, then give Bouncy and
// collisions.Contacts (which it reads) to whichever kinds should bounce. How
// hard that bounce is remains the entities' own business — see collisions.Mass
// and collisions.Restitution.
package elastic

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// Bouncy marks the entities this behavior bounces. One without it is still
// pushed out of an overlap, it just never rebounds off it.
type Bouncy struct{}

var _ world.Behavior = (*Behavior)(nil)

// Behavior rebounds Bouncy entities off whatever their Contacts say they struck.
type Behavior struct {
	query    *goke.Query
	vel      goke.Comp[world.Velocity]
	contacts goke.Comp[collisions.Contacts]
	mass     goke.OptComp[collisions.Mass]
}

func New() *Behavior { return &Behavior{} }

func (b *Behavior) Init(si *goke.SysInit) {
	si.RegComp[Bouncy]()
	b.query = si.NewQueryBuilder(&b.vel, &b.contacts).
		Optional(&b.mass).
		Include(goke.Include[Bouncy]()).
		Build()
}

func (b *Behavior) Update(*goke.CmdBuf, time.Duration) {
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		velocities := b.vel.Slice(cursor)
		contacts := b.contacts.Slice(cursor)
		masses := b.mass.Slice(cursor)

		for i := range cursor.IDs {
			var mass collisions.Mass
			if i < len(masses) {
				mass = masses[i]
			}
			inverse := 1 / mass.Weight()
			for _, c := range contacts[i].All() {
				if c.Impact == 0 {
					continue // they were already drawing apart, or only sensed
				}
				d := velocities[i].Delta()
				velocities[i].SetDelta(geom.NewVec(
					d.X+c.Impact*inverse*c.Normal.X,
					d.Y+c.Impact*inverse*c.Normal.Y,
				))
			}
		}
	}
}
