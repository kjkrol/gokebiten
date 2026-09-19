package collisions

import (
	"math"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/collide"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*NarrowPhase)(nil)

type NarrowPhase struct {
	space *gokg.Space

	touchQry    *goke.Query
	pos         goke.Comp[world.Position]
	vel         goke.Comp[world.Velocity]
	collision   goke.Comp[Collision]
	mass        goke.OptComp[Mass]
	restitution goke.OptComp[Restitution]
	contacts    goke.OptComp[Contacts]

	dynQry         *goke.Query
	dynPos         goke.Comp[world.Position]
	dynVel         goke.Comp[world.Velocity]
	dynMass        goke.OptComp[Mass]
	dynRestitution goke.OptComp[Restitution]
	dynContacts    goke.OptComp[Contacts]

	staticQuery       *goke.Query
	staticPos         goke.Comp[world.Position]
	staticRestitution goke.OptComp[Restitution]

	sensorIDs map[uid.UID64]struct{}
	staticIDs map[uid.UID64]struct{}

	// pending is what each side's velocity has already gained this tick, so a
	// second contact is resolved against the first rather than beside it.
	pending map[uid.UID64]geom.Vec

	// solver holds this tick's geometry, sides the identities and components
	// that go with it. The two are filled in lockstep, so a pair's index in
	// one names the same pair in the other.
	solver collide.Solver
	sides  []pairSides

	dynSeeded, staticSeeded bool
}

func NewNarrowPhase(space *gokg.Space) *NarrowPhase {
	return &NarrowPhase{
		space:     space,
		sensorIDs: make(map[uid.UID64]struct{}),
		staticIDs: make(map[uid.UID64]struct{}),
		pending:   make(map[uid.UID64]geom.Vec),
	}
}

func (n *NarrowPhase) Init(si *goke.SysInit) {
	n.touchQry = si.NewQueryBuilder(&n.pos, &n.vel, &n.collision).
		Optional(&n.mass, &n.restitution, &n.contacts).Build()
	n.dynQry = si.NewQueryBuilder(&n.dynPos, &n.dynVel).
		Optional(&n.dynMass, &n.dynRestitution, &n.dynContacts).Build()
	n.staticQuery = si.NewQueryBuilder(&n.staticPos).Optional(&n.staticRestitution).Build()
	sensorQry := si.NewQueryBuilder().Include(goke.Include[Sensor]()).Build()
	sensorQry.All()
	for sensorQry.Next() {
		for _, id := range sensorQry.Cursor().IDs {
			n.sensorIDs[id] = struct{}{}
		}
	}
	staticQry := si.NewQueryBuilder().Include(goke.Include[Static]()).Build()
	staticQry.All()
	for staticQry.Next() {
		for _, id := range staticQry.Cursor().IDs {
			n.staticIDs[id] = struct{}{}
		}
	}
}

// pairSides is what the solver deliberately knows nothing about: who the two
// boxes belong to, what else those entities carry, and whether the contact is
// only to be reported.
type pairSides struct {
	A, B   contactSide
	sensor bool
}

type contactSide struct {
	Entity uid.UID64
	Pos    *world.Position
	Vel    *world.Velocity
	// Mass is +Inf for a side with no Velocity — the whole bounce goes to the other one.
	Mass        float64
	Restitution float64
	// Contacts is nil unless this side asked to be told what it struck.
	Contacts *Contacts
}

// at is the i-th entry of an optional component's chunk slice, nil when the chunk carries none.
func at[T any](s []T, i int) *T {
	if i >= len(s) {
		return nil
	}
	return &s[i]
}

func (n *NarrowPhase) Update(_ *goke.CmdBuf, _ time.Duration) {
	const solverIterations = 16

	n.pair()
	n.solve(solverIterations)
}

func (n *NarrowPhase) isSensor(id uid.UID64) bool {
	_, ok := n.sensorIDs[id]
	return ok
}

func (n *NarrowPhase) isStatic(id uid.UID64) bool {
	_, ok := n.staticIDs[id]
	return ok
}

func (n *NarrowPhase) pair() {
	n.dynSeeded, n.staticSeeded = false, false
	n.solver.Reset()
	n.sides = n.sides[:0]

	n.touchQry.All()
	for n.touchQry.Next() {
		cursor := n.touchQry.Cursor()
		posSlice := n.pos.Slice(cursor)
		velSlice := n.vel.Slice(cursor)
		collisionSlice := n.collision.Slice(cursor)
		massSlice := n.mass.Slice(cursor)
		restitutionSlice := n.restitution.Slice(cursor)
		contactsSlice := n.contacts.Slice(cursor)
		for i, entityA := range cursor.IDs {
			p, v, c := &posSlice[i], &velSlice[i], &collisionSlice[i]
			if c.TouchingCount == 0 {
				continue
			}
			sideA := contactSide{
				Entity: entityA, Pos: p, Vel: v,
				Mass:        massOf(at(massSlice, i)),
				Restitution: restitutionOf(at(restitutionSlice, i)),
				Contacts:    at(contactsSlice, i),
			}

			for ti := uint8(0); ti < c.TouchingCount; ti++ {
				entityB := c.Touching[ti]
				if entityA.Index() >= entityB.Index() {
					continue
				}
				sideB, ok := n.resolveB(entityB)
				if !ok {
					continue
				}
				sensor := n.isSensor(entityA) || n.isSensor(entityB)
				n.solver.Add(collide.Pair{
					A: &p.AABB, B: &sideB.Pos.AABB,
					StaticB: sideB.Vel == nil,
					Sensor:  sensor,
				})
				n.sides = append(n.sides, pairSides{A: sideA, B: sideB, sensor: sensor})
			}

			c.clear()
		}
	}
}

// resolveB looks up id's Position (and Velocity, for a dynamic entity) via
// whichever of dynQry/staticQuery actually matches its archetype. Which one
// to try is decided up front by the Static tag, not by probing — Query.Seek
// is documented to bypass the archetype mask (it returns true for any
// existing entity), so a "try dynQry.Seek, fall back to staticQuery.Seek"
// probe would always succeed on the first try and read garbage bytes as a
// static entity's Velocity.
func (n *NarrowPhase) resolveB(id uid.UID64) (contactSide, bool) {
	if n.isStatic(id) {
		ok := n.staticSeeded && n.staticQuery.SeekH(id)
		if !ok {
			ok = n.staticQuery.Seek(id)
			n.staticSeeded = ok
		}
		if !ok {
			return contactSide{}, false
		}
		cur := n.staticQuery.Cursor()
		return contactSide{
			Entity: id, Pos: n.staticPos.At(cur),
			Mass:        math.Inf(1),
			Restitution: restitutionOf(n.staticRestitution.At(cur)),
		}, true
	}

	ok := n.dynSeeded && n.dynQry.SeekH(id)
	if !ok {
		ok = n.dynQry.Seek(id)
		n.dynSeeded = ok
	}
	if !ok {
		return contactSide{}, false
	}
	cur := n.dynQry.Cursor()
	return contactSide{
		Entity: id, Pos: n.dynPos.At(cur), Vel: n.dynVel.At(cur),
		Mass:        massOf(n.dynMass.At(cur)),
		Restitution: restitutionOf(n.dynRestitution.At(cur)),
		Contacts:    n.dynContacts.At(cur),
	}, true
}

func (n *NarrowPhase) solve(solverIterations int) {
	clear(n.pending)

	n.space.Resolve(&n.solver, solverIterations, func(i int, pen geom.Vec) {
		sides := &n.sides[i]

		normal, aligned := normalOf(pen)
		var impact float64
		if aligned && !sides.sensor {
			impact = n.exchange(sides.A, sides.B, normal)
		}
		// The normal points the way A leaves B, so B is told the opposite —
		// each side then reads its own contact as "this is where I go".
		if sides.A.Contacts != nil {
			sides.A.Contacts.add(sides.B.Entity, impact, normal)
		}
		if sides.B.Contacts != nil {
			sides.B.Contacts.add(sides.A.Entity, impact, geom.NewVec(-normal.X, -normal.Y))
		}
	})
	n.reindexMoved()
	n.space.Flush(nil)
}

// exchange is the impulse this contact trades along normal, booked against both
// sides so the ones after it start from what it left behind.
func (n *NarrowPhase) exchange(a, b contactSide, normal geom.Vec) float64 {
	deltaA, deltaB := n.velocityOf(a), n.velocityOf(b)
	impact := impactOf(a, b, deltaA, deltaB, normal)
	if impact == 0 {
		return 0
	}
	if inv := inverseMass(a); inv != 0 {
		n.pending[a.Entity] = plus(n.pending[a.Entity], impact*inv, normal)
	}
	if inv := inverseMass(b); inv != 0 {
		n.pending[b.Entity] = plus(n.pending[b.Entity], -impact*inv, normal)
	}
	return impact
}

// velocityOf is the side's velocity with this tick's earlier contacts folded in.
func (n *NarrowPhase) velocityOf(s contactSide) geom.Vec {
	return plus(deltaOf(s.Vel), 1, n.pending[s.Entity])
}

// plus is v plus scale times add.
func plus(v geom.Vec, scale float64, add geom.Vec) geom.Vec {
	return geom.NewVec(v.X+scale*add.X, v.Y+scale*add.Y)
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
			n.space.Reindex(sides.A.Entity, sides.A.Pos.AABB)
		}
		if movedB {
			n.space.Reindex(sides.B.Entity, sides.B.Pos.AABB)
		}
	})
}
