package collisions

import (
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
	space       *gokg.Space
	handler     CollisionHandler
	hitDuration time.Duration

	hitQry     *goke.Query
	pos        goke.Comp[world.Position]
	vel        goke.Comp[world.Velocity]
	hitTag     goke.Comp[Hit]
	collision  goke.Comp[Collision]
	hitExpires goke.OptComp[HitExpires]

	dynQry *goke.Query
	dynPos goke.Comp[world.Position]
	dynVel goke.Comp[world.Velocity]

	staticQuery *goke.Query
	staticPos   goke.Comp[world.Position]

	sensorIDs map[uid.UID64]struct{}
	staticIDs map[uid.UID64]struct{}

	removeEditor *goke.Editor

	// solver holds this tick's geometry, sides the identities and components
	// that go with it. The two are filled in lockstep, so a pair's index in
	// one names the same pair in the other.
	solver collide.Solver
	sides  []pairSides

	dynSeeded, staticSeeded bool

	confirmed map[uid.UID64]bool
}

func NewNarrowPhase(space *gokg.Space, handler CollisionHandler, hitDuration time.Duration) *NarrowPhase {
	return &NarrowPhase{
		space: space, handler: handler, hitDuration: hitDuration,
		confirmed: make(map[uid.UID64]bool),
		sensorIDs: make(map[uid.UID64]struct{}),
		staticIDs: make(map[uid.UID64]struct{}),
	}
}

func (n *NarrowPhase) Init(si *goke.SysInit) {
	n.hitQry = si.NewQueryBuilder(&n.pos, &n.vel, &n.hitTag, &n.collision).Optional(&n.hitExpires).Build()
	n.dynQry = si.NewQueryBuilder(&n.dynPos, &n.dynVel).Build()
	n.staticQuery = si.NewQueryBuilder(&n.staticPos).Build()
	if init, ok := n.handler.(Initializer); ok {
		init.Init(si)
	}
	n.removeEditor = n.hitQry.NewEditorBuilder().Remove(goke.Remove[Hit]()).Build()
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
}

func (n *NarrowPhase) Update(cb *goke.CmdBuf, _ time.Duration) {
	const solverIterations = 16
	clear(n.confirmed)

	n.pair()
	n.solve(cb, solverIterations)
	n.finalizeHitTags(cb)
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

	n.hitQry.All()
	for n.hitQry.Next() {
		cursor := n.hitQry.Cursor()
		posSlice := n.pos.Slice(cursor)
		velSlice := n.vel.Slice(cursor)
		collisionSlice := n.collision.Slice(cursor)
		for i, entityA := range cursor.IDs {
			p, v, c := &posSlice[i], &velSlice[i], &collisionSlice[i]
			n.confirmed[entityA] = false

			for ti := uint8(0); ti < c.TouchingCount; ti++ {
				entityB := c.Touching[ti]
				if entityA.Index() >= entityB.Index() {
					continue
				}
				posB, velB, ok := n.resolveB(entityB)
				if !ok {
					continue
				}
				n.solver.Add(collide.Pair{
					A: &p.AABB, B: &posB.AABB,
					StaticB: velB == nil,
					Sensor:  n.isSensor(entityA) || n.isSensor(entityB),
				})
				n.sides = append(n.sides, pairSides{
					A:      contactSide{Entity: entityA, Pos: p, Vel: v},
					B:      contactSide{Entity: entityB, Pos: posB, Vel: velB},
					sensor: n.isSensor(entityA) || n.isSensor(entityB),
				})
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
func (n *NarrowPhase) resolveB(id uid.UID64) (pos *world.Position, vel *world.Velocity, ok bool) {
	if n.isStatic(id) {
		ok = n.staticSeeded && n.staticQuery.SeekH(id)
		if !ok {
			ok = n.staticQuery.Seek(id)
			n.staticSeeded = ok
		}
		if !ok {
			return nil, nil, false
		}
		return n.staticPos.At(n.staticQuery.Cursor()), nil, true
	}

	ok = n.dynSeeded && n.dynQry.SeekH(id)
	if !ok {
		ok = n.dynQry.Seek(id)
		n.dynSeeded = ok
	}
	if !ok {
		return nil, nil, false
	}
	cur := n.dynQry.Cursor()
	return n.dynPos.At(cur), n.dynVel.At(cur), true
}

func (n *NarrowPhase) solve(cb *goke.CmdBuf, solverIterations int) {
	n.space.Resolve(&n.solver, solverIterations, func(i int, pen geom.Vec) {
		sides := &n.sides[i]

		// Confirmation is about "something struck me" and holds for a sensor
		// contact too; only the physical reaction is withheld from one.
		if _, ok := n.confirmed[sides.A.Entity]; ok {
			n.confirmed[sides.A.Entity] = true
		}
		if _, ok := n.confirmed[sides.B.Entity]; ok {
			n.confirmed[sides.B.Entity] = true
		}
		if sides.sensor || n.handler == nil {
			return
		}
		n.handler.OnCollision(cb, CollisionEvent{
			EntityA: sides.A.Entity, EntityB: sides.B.Entity,
			PosA: sides.A.Pos, PosB: sides.B.Pos,
			VelA: sides.A.Vel, VelB: sides.B.Vel,
			Penetration: pen,
		})
	})
	n.reindexMoved()
	n.space.Flush(nil)
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

func (n *NarrowPhase) finalizeHitTags(cb *goke.CmdBuf) {
	now := time.Now()
	defaultUntil := now.Add(n.hitDuration)

	n.hitQry.All()
	for n.hitQry.Next() {
		cursor := n.hitQry.Cursor()
		tags := n.hitTag.Slice(cursor)
		hasOverride := n.hitExpires.Present(cursor)
		var overrides []HitExpires
		if hasOverride {
			overrides = n.hitExpires.Slice(cursor)
		}
		buf := n.hitQry.BeginMigrate(cb)
		for i, id := range cursor.IDs {
			confirmedThisTick, tracked := n.confirmed[id]
			if !tracked {
				continue
			}
			if confirmedThisTick {
				until := defaultUntil
				if hasOverride {
					until = now.Add(overrides[i].Duration)
				}
				tags[i].SetExpiresAt(until)
				continue
			}
			if tags[i].HasExpiry() {
				continue
			}
			buf.Add(id)
		}
		buf.Commit(n.removeEditor)
	}
}
