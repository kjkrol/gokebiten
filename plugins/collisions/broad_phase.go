package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
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
	base      goke.Comp[world.Base]
	collision goke.Comp[Collision]

	// host runs the Each behaviors registered with the plugin, inside this pass.
	host *plugin.EachHost[Struck]

	// walking is the chunk the host is being run over — what struckAt reads.
	// The func is bound once for the same reason probe.onFound is.
	walking struct {
		ids        []uid.UID64
		collisions []Collision
	}
	struckAt func(i int) Struck

	probe neighbourProbe
}

// probe is the state one entity's neighbour search runs against, held on the
// system rather than captured fresh per entity so the callback handed to
// Space.Neighbours can be bound once — the same reason raycast.View binds its
// collector once. A closure built inside the loop would escape to the heap on
// every entity, every tick.
type neighbourProbe struct {
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
	return newBroadPhase(space, margin, &plugin.EachHost[Struck]{})
}

func newBroadPhase(space *gokg.Space, margin float64, host *plugin.EachHost[Struck]) *BroadPhase {
	b := &BroadPhase{space: space, margin: margin, host: host}
	b.struckAt = b.struck
	b.probe.onFound = b.probe.found
	return b
}

func (b *BroadPhase) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&b.base, &b.collision)
	b.host.Bind(qb)
	b.query = qb.Build()
}

func (b *BroadPhase) Update(cb *goke.CmdBuf, d time.Duration) {
	t := plugin.Tick{Cmd: cb, Now: time.Now(), Dt: d}
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		bases := b.base.Slice(cursor)
		collisionSlice := b.collision.Slice(cursor)

		// Hosted behaviors go first: what they read is last tick's contacts,
		// which the loop below is about to clear.
		b.walking.ids, b.walking.collisions = cursor.IDs, collisionSlice
		b.host.Run(t, cursor, b.struckAt)
		for i, entityA := range cursor.IDs {
			collisionSlice[i].clearContacts()
			b.probe.self = entityA
			b.probe.touching = &collisionSlice[i]
			b.probe.box = bases[i].Pos.AABB

			b.space.Neighbours(&b.probe.box, b.margin, CanCollide, b.probe.onFound)
		}
	}
}

// struck is what the hosted behaviors are told about the i-th entity of the chunk being walked.
func (b *BroadPhase) struck(i int) Struck {
	return Struck{ID: b.walking.ids[i], Contacts: b.walking.collisions[i].Contacts()}
}

// found records one candidate. Whether the neighbour takes part in collisions
// at all is the index's answer now, not a lookup here — only the prober itself
// still has to be filtered out. Which wrapped image was hit makes no
// difference either; the narrow phase works that out again from current
// geometry.
func (p *neighbourProbe) found(other uid.UID64, _ plane.FragPosition) {
	if other == p.self {
		return
	}
	p.touching.addTouching(other)
}
