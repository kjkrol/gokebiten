package collision

import (
	"github.com/kjkrol/aabbworld/collide"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*NarrowPhase)(nil)

type NarrowPhase struct {
	space *aabbworld.Space

	found *Candidates

	// lookup resolves both sides of a candidate from their ids.
	lookup         *goke.Query
	lookupBase     goke.Comp[world.Base]
	lookupCollider goke.Comp[Collider]
	lookupPhysics  goke.OptComp[Physics]
	lookupHot      bool

	// narrow holds this tick's geometry and sides who it belongs to, pair for pair by index.
	narrow collide.NarrowPhase
	sides  []pairSides

	// host runs the Between behaviors registered with the plugin, inside this pass.
	host *plugin.PairHost[Meeting]

	tracked func(t plugin.Tick, id uid.UID64, inside bool)
	whose   func(i int) (uid.UID64, uid.UID64)
}

// sought is the one query the narrow phase offers its hosted behaviors.
const sought = 0

// NewNarrowPhase builds the narrow phase over space, testing the pairs found holds.
func NewNarrowPhase(space *aabbworld.Space, found *Candidates) *NarrowPhase {
	return newNarrowPhase(space, found, &plugin.PairHost[Meeting]{})
}

func newNarrowPhase(space *aabbworld.Space, found *Candidates, host *plugin.PairHost[Meeting]) *NarrowPhase {
	n := &NarrowPhase{space: space, found: found, host: host, tracked: func(plugin.Tick, uid.UID64, bool) {}}
	n.whose = n.entitiesOf
	return n
}

func (n *NarrowPhase) Init(si *goke.SysInit) {
	seek := si.NewQueryBuilder(&n.lookupBase, &n.lookupCollider).Optional(&n.lookupPhysics)
	n.host.Bind(seek)
	n.lookup = seek.Build()
}

// pairSides is who the two boxes of a pair belong to, what they carry,
// and how the contact went once confirmed.
type pairSides struct {
	A, B         contactSide
	tagsA, tagsB uint64

	confirmed bool
	impact    float64
	normal    geom.Vec
}

// detectOnly reports a pair in which either side takes no part in the physical world.
func (p *pairSides) detectOnly() bool { return p.A.Physics == nil || p.B.Physics == nil }

type contactSide struct {
	Entity   uid.UID64
	Base     *world.Base
	Collider *Collider
	// Physics is nil for a side that is only ever detected.
	Physics *Physics
}

// immovable reports a side the narrow phase must not push: nothing shifts an infinite mass.
func (s contactSide) immovable() bool { return s.Physics != nil && s.Physics.Immovable() }

func (n *NarrowPhase) Update(cb *goke.CmdBuf, d time.Duration) {
	const solverIterations = 16

	n.pair()
	n.solve(plugin.Tick{Cmd: cb, Now: time.Now(), Dt: d}, solverIterations)
}

func (n *NarrowPhase) pair() {
	n.lookupHot = false
	n.narrow.Reset()
	n.sides = n.sides[:0]

	for _, c := range n.found.All() {
		sideA, tagsA, ok := n.resolve(c.A)
		if !ok {
			continue
		}
		sideB, tagsB, ok := n.resolve(c.B)
		if !ok {
			continue
		}
		sides := pairSides{A: sideA, B: sideB, tagsA: tagsA, tagsB: tagsB}
		n.narrow.Add(collide.Pair{
			A: &sideA.Base.Pos.AABB, B: &sideB.Base.Pos.AABB,
			KeyA: c.A.Index(), KeyB: c.B.Index(),
			StaticA: sideA.immovable(),
			StaticB: sideB.immovable(),
			Sensor:  sides.detectOnly(),
		})
		n.sides = append(n.sides, sides)
	}
}

// resolve looks one side of a candidate up, refusing one that no longer carries a Collider.
func (n *NarrowPhase) resolve(id uid.UID64) (contactSide, uint64, bool) {
	ok := n.lookupHot && n.lookup.SeekH(id)
	if !ok {
		ok = n.lookup.Seek(id)
		n.lookupHot = ok
	}
	var collider *Collider
	if ok {
		collider = n.lookupCollider.At(n.lookup.Cursor())
	}
	if collider == nil {
		n.space.SetCapabilities(id, aabbworld.Plain)
		return contactSide{}, 0, false
	}
	cur := n.lookup.Cursor()
	return contactSide{
		Entity: id, Base: n.lookupBase.At(cur),
		Collider: collider, Physics: n.lookupPhysics.At(cur),
	}, n.host.At(sought, cur), true
}

// entitiesOf names the two entities of the i-th pair handed to the narrow phase.
func (n *NarrowPhase) entitiesOf(i int) (uid.UID64, uid.UID64) {
	return n.sides[i].A.Entity, n.sides[i].B.Entity
}

func (n *NarrowPhase) solve(t plugin.Tick, solverIterations int) {
	n.narrow.Separate(n.space, solverIterations, func(i int, pen geom.Vec) {
		sides := &n.sides[i]

		normal, aligned := normalOf(pen)
		var impact float64
		if aligned && !sides.detectOnly() {
			impact = bounce(sides.A, sides.B, normal)
		}
		sides.confirmed, sides.impact, sides.normal = true, impact, normal

		sides.A.Collider.addContact(sides.B.Entity, impact, normal)
		sides.B.Collider.addContact(sides.A.Entity, impact, geom.NewVec(-normal.X, -normal.Y))
	}, n.whose)
	for _, id := range n.narrow.Left() {
		n.tracked(t, id, false)
	}

	for i := range n.sides {
		if s := &n.sides[i]; s.confirmed {
			n.host.DispatchEitherWay(t, s.tagsA, s.tagsB,
				Meeting{Self: s.A.Entity, Other: s.B.Entity, Impact: s.impact, Normal: s.normal},
				Meeting{Self: s.B.Entity, Other: s.A.Entity, Impact: s.impact, Normal: geom.NewVec(-s.normal.X, -s.normal.Y)})
		}
	}
	n.space.Flush(nil)
}

// bounce trades the contact's impulse between two physical sides and returns it.
func bounce(a, b contactSide, normal geom.Vec) float64 {
	deltaA, deltaB := a.Base.Vel.Delta(), b.Base.Vel.Delta()
	impact := impactOf(*a.Physics, *b.Physics, deltaA, deltaB, normal)
	if impact == 0 {
		return 0
	}
	if inv := inverseMass(*a.Physics); inv != 0 {
		a.Base.Vel.SetDelta(geom.NewVec(deltaA.X+impact*inv*normal.X, deltaA.Y+impact*inv*normal.Y))
	}
	if inv := inverseMass(*b.Physics); inv != 0 {
		b.Base.Vel.SetDelta(geom.NewVec(deltaB.X-impact*inv*normal.X, deltaB.Y-impact*inv*normal.Y))
	}
	return impact
}
