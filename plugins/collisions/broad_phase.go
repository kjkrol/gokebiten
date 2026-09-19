package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

// CanCollide is the capability an entity needs to be offered as a collision
// candidate, so a probe can ask the index for those rather than for everything
// and reject most of what comes back. Give it with [Collidable].
//
// Bit 0 is gokg's Plain; a second subsystem wanting the same treatment picks
// its own bit.
const CanCollide gokg.Capability = 1 << 1

var _ goke.System = (*BroadPhase)(nil)

// BroadPhase records, for every collidable entity, who is close enough to be worth
// a real overlap test this tick, and drops what that entity struck the tick before.
type BroadPhase struct {
	space     *gokg.Space
	margin    float64
	query     *goke.Query
	pos       goke.Comp[world.Position]
	vel       goke.Comp[world.Velocity]
	collision goke.Comp[Collision]
	contacts  goke.OptComp[Contacts]

	probe probe
}

// probe is the state one entity's neighbour search runs against, held on the
// system rather than captured fresh per entity so the callback handed to
// Space.Neighbours can be bound once — the same reason raycast.View binds its
// collector once. A closure built inside the loop would escape to the heap on
// every entity, every tick.
type probe struct {
	box      plane.AABB
	self     uid.UID64
	touching *Collision
	onFound  func(uid.UID64, plane.FragPosition)
}

// NewBroadPhase builds the broad phase over space, reaching margin past each
// entity's own box when it probes for neighbours. margin has to cover how far
// a pair can close between two broad phases — see world.Plugin.MaxStep — and
// nothing more: every extra unit is area the spatial index has to scan and
// candidates the narrow phase then has to reject.
func NewBroadPhase(space *gokg.Space, margin float64) *BroadPhase {
	b := &BroadPhase{space: space, margin: margin}
	b.probe.onFound = b.probe.found
	return b
}

func (b *BroadPhase) Init(si *goke.SysInit) {
	b.query = si.NewQueryBuilder(&b.pos, &b.vel, &b.collision).Optional(&b.contacts).Build()
}

func (b *BroadPhase) Update(_ *goke.CmdBuf, _ time.Duration) {
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		posSlice := b.pos.Slice(cursor)
		collisionSlice := b.collision.Slice(cursor)
		contactsSlice := b.contacts.Slice(cursor)
		for i, entityA := range cursor.IDs {
			if c := at(contactsSlice, i); c != nil {
				c.clear()
			}
			b.probe.self = entityA
			b.probe.touching = &collisionSlice[i]
			b.probe.box = posSlice[i].AABB

			b.space.Neighbours(&b.probe.box, b.margin, CanCollide, b.probe.onFound)
		}
	}
}

// found records one candidate. Whether the neighbour takes part in collisions
// at all is the index's answer now, not a lookup here — only the prober itself
// still has to be filtered out. Which wrapped image was hit makes no
// difference either; the narrow phase works that out again from current
// geometry.
func (p *probe) found(other uid.UID64, _ plane.FragPosition) {
	if other == p.self {
		return
	}
	p.touching.addTouching(other)
}
