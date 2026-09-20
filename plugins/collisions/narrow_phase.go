package collisions

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/collide"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*NarrowPhase)(nil)

type NarrowPhase struct {
	space *gokg.Space

	// touchQry walks side A; lookup seeks side B. They are the same shape twice
	// over because a Query has one cursor, and seeking B on the query that is
	// being iterated would lose A's place.
	touchQry  *goke.Query
	base      goke.Comp[world.Base]
	collision goke.Comp[Collision]
	physics   goke.OptComp[Physics]

	lookup          *goke.Query
	lookupBase      goke.Comp[world.Base]
	lookupCollision goke.Comp[Collision]
	lookupPhysics   goke.OptComp[Physics]
	lookupHot       bool

	// solver holds this tick's geometry, sides the identities and components
	// that go with it. The two are filled in lockstep, so a pair's index in
	// one names the same pair in the other.
	solver collide.Solver
	sides  []pairSides

	// host runs the Between behaviors registered with the plugin, inside this pass.
	host *plugin.PairHost[Meeting]
}

// The two queries the narrow phase offers its hosted behaviors, by index.
const (
	walked = iota // side A, a chunk at a time
	sought        // side B, one entity at a time
)

func NewNarrowPhase(space *gokg.Space) *NarrowPhase {
	return newNarrowPhase(space, &plugin.PairHost[Meeting]{})
}

func newNarrowPhase(space *gokg.Space, host *plugin.PairHost[Meeting]) *NarrowPhase {
	return &NarrowPhase{space: space, host: host}
}

func (n *NarrowPhase) Init(si *goke.SysInit) {
	walk := si.NewQueryBuilder(&n.base, &n.collision).Optional(&n.physics)
	seek := si.NewQueryBuilder(&n.lookupBase, &n.lookupCollision).Optional(&n.lookupPhysics)
	n.host.Bind(walk, seek)
	n.touchQry, n.lookup = walk.Build(), seek.Build()
}

// pairSides is what the solver deliberately knows nothing about: who the two
// boxes belong to, what else those entities carry — and, once the solver has
// confirmed the contact, how it went.
type pairSides struct {
	A, B         contactSide
	tagsA, tagsB uint64

	confirmed bool
	impact    float64
	normal    geom.Vec
}

// detectOnly reports a pair the solver is only to report: one side or the other
// takes no part in the physical world.
func (p *pairSides) detectOnly() bool { return p.A.Physics == nil || p.B.Physics == nil }

type contactSide struct {
	Entity    uid.UID64
	Base      *world.Base
	Collision *Collision
	// Physics is nil for a side that is only ever detected.
	Physics *Physics
}

// immovable reports a side the solver must not push: nothing shifts an infinite mass.
func (s contactSide) immovable() bool { return s.Physics != nil && s.Physics.Immovable() }

// at is the i-th entry of an optional component's chunk slice, nil when the chunk carries none.
func at[T any](s []T, i int) *T {
	if i >= len(s) {
		return nil
	}
	return &s[i]
}

func (n *NarrowPhase) Update(cb *goke.CmdBuf, d time.Duration) {
	const solverIterations = 16

	n.pair()
	n.solve(plugin.Tick{Cmd: cb, Now: time.Now(), Dt: d}, solverIterations)
}

func (n *NarrowPhase) pair() {
	n.lookupHot = false
	n.solver.Reset()
	n.sides = n.sides[:0]

	n.touchQry.All()
	for n.touchQry.Next() {
		cursor := n.touchQry.Cursor()
		bases := n.base.Slice(cursor)
		collisionSlice := n.collision.Slice(cursor)
		physicsSlice := n.physics.Slice(cursor)
		chunkTags := n.host.InChunk(walked, cursor)
		for i, entityA := range cursor.IDs {
			c := &collisionSlice[i]
			if c.TouchingCount == 0 {
				continue
			}
			sideA := contactSide{Entity: entityA, Base: &bases[i], Collision: c, Physics: at(physicsSlice, i)}

			for ti := uint8(0); ti < c.TouchingCount; ti++ {
				entityB := c.Touching[ti]
				if entityA.Index() >= entityB.Index() {
					continue
				}
				sideB, tagsB, ok := n.resolveB(entityB)
				if !ok {
					continue
				}
				sides := pairSides{A: sideA, B: sideB, tagsA: chunkTags, tagsB: tagsB}
				n.solver.Add(collide.Pair{
					A: &sideA.Base.Pos.AABB, B: &sideB.Base.Pos.AABB,
					KeyA: entityA.Index(), KeyB: entityB.Index(),
					StaticA: sideA.immovable(),
					StaticB: sideB.immovable(),
					Sensor:  sides.detectOnly(),
				})
				n.sides = append(n.sides, sides)
			}

			c.clear()
		}
	}
}

// resolveB looks id up for side B. Query.Seek finds any living entity whatever
// the query's mask, so a candidate that turns out to carry no Collision is
// refused here rather than trusted to the index.
func (n *NarrowPhase) resolveB(id uid.UID64) (contactSide, uint64, bool) {
	ok := n.lookupHot && n.lookup.SeekH(id)
	if !ok {
		ok = n.lookup.Seek(id)
		n.lookupHot = ok
	}
	if !ok {
		return contactSide{}, 0, false
	}
	cur := n.lookup.Cursor()
	collision := n.lookupCollision.At(cur)
	if collision == nil {
		return contactSide{}, 0, false
	}
	return contactSide{
		Entity: id, Base: n.lookupBase.At(cur),
		Collision: collision, Physics: n.lookupPhysics.At(cur),
	}, n.host.At(sought, cur), true
}

func (n *NarrowPhase) solve(t plugin.Tick, solverIterations int) {
	n.space.Resolve(&n.solver, solverIterations, func(i int, pen geom.Vec) {
		sides := &n.sides[i]

		normal, aligned := normalOf(pen)
		var impact float64
		if aligned && !sides.detectOnly() {
			impact = bounce(sides.A, sides.B, normal)
		}
		sides.confirmed, sides.impact, sides.normal = true, impact, normal

		// The normal points the way A leaves B, so B is told the opposite —
		// each side then reads its own contact as "this is where I went".
		sides.A.Collision.addContact(sides.B.Entity, impact, normal)
		sides.B.Collision.addContact(sides.A.Entity, impact, geom.NewVec(-normal.X, -normal.Y))
	})
	n.reindexMoved()

	// Behaviors run once the solver is done and every box is back in the index
	// where it came to rest — so one that despawns an entity queues its removal
	// after the last reindex, not before it.
	for i := range n.sides {
		if s := &n.sides[i]; s.confirmed {
			n.host.DispatchEitherWay(t, s.tagsA, s.tagsB,
				Meeting{Self: s.A.Entity, Other: s.B.Entity, Impact: s.impact, Normal: s.normal},
				Meeting{Self: s.B.Entity, Other: s.A.Entity, Impact: s.impact, Normal: geom.NewVec(-s.normal.X, -s.normal.Y)})
		}
	}
	n.space.Flush(nil)
}

// bounce trades the contact's impulse between two physical sides there and
// then, so the next contact either of them is in starts from what this one left.
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

// reindexMoved tells the spatial index where the boxes the solver pushed
// finally came to rest. Only the settled position matters, so one update per
// moved side replaces the one per side per iteration the solver used to queue
// — which, at solverIterations times the contact count, could outrun
// OpsBufferSize and block on the queue with nothing left to drain it.
func (n *NarrowPhase) reindexMoved() {
	n.solver.VisitMoved(func(i int, movedA, movedB bool) {
		sides := &n.sides[i]
		if movedA {
			n.space.Reindex(sides.A.Entity, sides.A.Base.Pos.AABB)
		}
		if movedB {
			n.space.Reindex(sides.B.Entity, sides.B.Base.Pos.AABB)
		}
	})
}
