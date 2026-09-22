package collision

import (
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/collide"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

var _ goke.System = (*Detector)(nil)

// solverIterations caps the passes one tick spends separating chained overlaps.
const solverIterations = 16

// Detector runs one tick of collisions: what each entity struck last tick, who may touch now,
// who really overlaps, the bounce, the push apart, and the contacts left behind for behaviors.
type Detector struct {
	space  *aabbworld.Space
	engine collide.Engine

	// walk is the pass over every Collider: its Each behaviors and its mark in the index.
	walk     *goke.Query
	base     goke.Comp[world.Base]
	collider goke.Comp[Collider]
	each     *plugin.EachHost[Struck]
	walking  struct {
		ids       []uid.UID64
		colliders []Collider
	}
	struckAt func(i int) Struck

	// lookup resolves both sides of a candidate from their ids.
	lookup         *goke.Query
	lookupBase     goke.Comp[world.Base]
	lookupCollider goke.Comp[Collider]
	lookupPhysics  goke.OptComp[Physics]
	lookupHot      bool

	// sides is who each pair handed to the engine belongs to, and how its contact went.
	sides   []pairSides
	between *plugin.PairHost[Meeting]
	tracked func(t plugin.Tick, id uid.UID64, inside bool)
	shapes  ShapeTest

	tick      plugin.Tick
	resolve   collide.Resolve
	touch     collide.Touch
	onContact func(i int, pen geom.Vec)
}

// sought is the one query the detector offers its hosted behaviors.
const sought = 0

// NewDetector builds the collision detector over space.
func NewDetector(space *aabbworld.Space) *Detector {
	return newDetector(space, &plugin.PairHost[Meeting]{}, &plugin.EachHost[Struck]{}, nil)
}

func newDetector(space *aabbworld.Space, between *plugin.PairHost[Meeting], each *plugin.EachHost[Struck], shapes ShapeTest) *Detector {
	d := &Detector{space: space, between: between, each: each, shapes: shapes, tracked: func(plugin.Tick, uid.UID64, bool) {}}
	d.struckAt = d.struck
	d.resolve = d.resolvePair
	d.onContact = d.contact
	if shapes != nil {
		d.touch = d.shapesTouch
	}
	return d
}

func (d *Detector) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&d.base, &d.collider)
	d.each.Bind(qb)
	d.walk = qb.Build()

	seek := si.NewQueryBuilder(&d.lookupBase, &d.lookupCollider).Optional(&d.lookupPhysics)
	d.between.Bind(seek)
	d.lookup = seek.Build()
}

func (d *Detector) Update(cb *goke.CmdBuf, dt time.Duration) {
	d.tick = plugin.Tick{Cmd: cb, Now: time.Now(), Dt: dt}
	d.mark()

	d.sides = d.sides[:0]
	d.lookupHot = false
	d.engine.Tick(d.space, world.StepReach, aabbworld.CanCollide, solverIterations, d.resolve, d.touch, d.onContact)
	for _, id := range d.engine.Left() {
		d.tracked(d.tick, id, false)
	}

	for i := range d.sides {
		if s := &d.sides[i]; s.confirmed {
			d.between.DispatchEitherWay(d.tick, s.tagsA, s.tagsB,
				Meeting{Self: s.A.Entity, Other: s.B.Entity, Impact: s.impact, Normal: s.normal},
				Meeting{Self: s.B.Entity, Other: s.A.Entity, Impact: s.impact, Normal: geom.NewVec(-s.normal.X, -s.normal.Y)})
		}
	}
	d.space.Flush(nil)
}

// mark runs the Each behaviors over every Collider, clears its contacts and indexes new ones.
func (d *Detector) mark() {
	marked := false
	d.walk.All()
	for d.walk.Next() {
		cursor := d.walk.Cursor()
		colliders := d.collider.Slice(cursor)

		d.walking.ids, d.walking.colliders = cursor.IDs, colliders
		d.each.Run(d.tick, cursor, d.struckAt)
		for i, id := range cursor.IDs {
			c := &colliders[i]
			c.clearContacts()
			if !c.Indexed {
				d.space.SetCapabilities(id, aabbworld.CanCollide)
				c.Indexed, marked = true, true
			}
		}
	}
	if marked {
		d.space.Flush(nil)
	}
}

// struck is what the hosted behaviors are told about the i-th entity of the chunk being walked.
func (d *Detector) struck(i int) Struck {
	return Struck{ID: d.walking.ids[i], Contacts: d.walking.colliders[i].Contacts()}
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

// immovable reports a side the engine must not push: nothing shifts an infinite mass.
func (s contactSide) immovable() bool { return s.Physics != nil && s.Physics.Immovable() }

func (s contactSide) body(sensor bool) collide.Body {
	return collide.Body{Box: &s.Base.Pos.AABB, Static: s.immovable(), Sensor: sensor}
}

// resolvePair looks both sides of a candidate up and keeps them beside the engine's pair.
func (d *Detector) resolvePair(a, b uid.UID64) (collide.Body, collide.Body, bool) {
	sideA, tagsA, ok := d.resolveSide(a)
	if !ok {
		return collide.Body{}, collide.Body{}, false
	}
	sideB, tagsB, ok := d.resolveSide(b)
	if !ok {
		return collide.Body{}, collide.Body{}, false
	}
	sides := pairSides{A: sideA, B: sideB, tagsA: tagsA, tagsB: tagsB}
	d.sides = append(d.sides, sides)
	sensor := sides.detectOnly()
	return sideA.body(sensor), sideB.body(sensor), true
}

// resolveSide looks one side of a candidate up, refusing one that no longer carries a Collider.
func (d *Detector) resolveSide(id uid.UID64) (contactSide, uint64, bool) {
	ok := d.lookupHot && d.lookup.SeekH(id)
	if !ok {
		ok = d.lookup.Seek(id)
		d.lookupHot = ok
	}
	var collider *Collider
	if ok {
		collider = d.lookupCollider.At(d.lookup.Cursor())
	}
	if collider == nil {
		d.space.SetCapabilities(id, aabbworld.Plain)
		return contactSide{}, 0, false
	}
	cur := d.lookup.Cursor()
	return contactSide{
		Entity: id, Base: d.lookupBase.At(cur),
		Collider: collider, Physics: d.lookupPhysics.At(cur),
	}, d.between.At(sought, cur), true
}

// shapesTouch asks the plugin's ShapeTest about the i-th pair.
func (d *Detector) shapesTouch(i int, pen geom.Vec) (geom.Vec, bool) {
	s := &d.sides[i]
	return d.shapes(d.tick, Contactee{ID: s.A.Entity, Base: s.A.Base}, Contactee{ID: s.B.Entity, Base: s.B.Base}, pen)
}

// contact settles the i-th pair's confirmed contact: the bounce, and a Contact on each side.
func (d *Detector) contact(i int, pen geom.Vec) {
	sides := &d.sides[i]

	normal, aligned := normalOf(pen)
	var impact float64
	if aligned && !sides.detectOnly() {
		impact = bounce(sides.A, sides.B, normal)
	}
	sides.confirmed, sides.impact, sides.normal = true, impact, normal

	sides.A.Collider.addContact(sides.B.Entity, impact, normal)
	sides.B.Collider.addContact(sides.A.Entity, impact, geom.NewVec(-normal.X, -normal.Y))
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
