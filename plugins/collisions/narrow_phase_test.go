package collisions_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

func runNarrowPhase(t *testing.T, space *gokg.Space, setup ...goke.System) *goke.ECS {
	t.Helper()
	ecs := goke.New()
	ecs.Setup(setup...)

	np := collisions.NewNarrowPhase(space)
	handle := ecs.RegSys(np)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)
	return ecs
}

// findPos scans q for id specifically and returns its Position — q may also
// match other entities, so this never trusts "the first thing the query
// finds" the way a bare range over q.Next() would.
func findPos(t *testing.T, q *goke.Query, posComp goke.Comp[world.Position], id uid.UID64) world.Position {
	t.Helper()
	q.All()
	for q.Next() {
		cur := q.Cursor()
		slice := posComp.Slice(cur)
		for i, gotID := range cur.IDs {
			if gotID == id {
				return slice[i]
			}
		}
	}
	t.Fatalf("entity %v not found by the query", id)
	return world.Position{}
}

func TestNarrowPhase_DynamicDynamic_Overlap(t *testing.T) {
	space := testSpace(t)
	var idA, idB uid.UID64
	var posComp goke.Comp[world.Position]
	var q *goke.Query

	runNarrowPhase(t, space, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posA goke.Comp[world.Position]
		var velA goke.Comp[world.Velocity]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &collA)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10) // plenty of margin from the world edge
		idA = fa.IDs[0]

		var posB goke.Comp[world.Position]
		var velB goke.Comp[world.Velocity]
		fb := si.NewFactory(&posB, &velB)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10) // overlaps A by 5px on X
		idB = fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		posComp = posA
		q = si.NewQueryBuilder(&posA).Include(goke.Include[collisions.Collision]()).Build()
	}})

	if p := findPos(t, q, posComp, idA); p.TopLeft.X == 100 {
		t.Error("expected A's position to have moved apart from B")
	}
}

func TestNarrowPhase_DynamicStatic_OnlyDynamicMoves(t *testing.T) {
	space := testSpace(t)
	var posA, posB goke.Comp[world.Position]
	var idA, idB uid.UID64
	var qA, qB *goke.Query
	var staticEditor *goke.Editor
	var contactsComp goke.Comp[collisions.Contacts]
	var contactsQ *goke.Query

	seed := goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var velA goke.Comp[world.Velocity]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &collA, &contactsComp)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10)   // plenty of margin from the world edge
		velA.Slice(&fa.Cursor)[0].SetDelta(geom.NewVec(4, 0)) // closing on the wall
		idA = fa.IDs[0]

		fb := si.NewFactory(&posB) // no Velocity -> must be tagged Static (see addStatic below)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10)
		idB = fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		qA = si.NewQueryBuilder(&posA).Include(goke.Include[world.Velocity]()).Build()
		qB = si.NewQueryBuilder(&posB).Exclude(goke.Exclude[world.Velocity]()).Build()
		contactsQ = si.NewQueryBuilder(&contactsComp).Build()
	}}

	// Static is a zero-size tag: Factory/Track reject zero-size data
	// columns, so — same as Sensor — it's added via an Editor migration in
	// a second Setup-phase system run after the seed.
	var staticComp goke.Comp[collisions.Static]
	addStatic := goke.SystemFn{
		OnInit: func(si *goke.SysInit) {
			staticEditor = qB.NewEditorBuilder(&staticComp).Build()
		},
		OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
			qB.All()
			for qB.Next() {
				buf := qB.BeginMigrate(cb)
				for _, id := range qB.Cursor().IDs {
					buf.Add(id)
				}
				buf.Commit(staticEditor)
			}
		},
	}

	runNarrowPhase(t, space, seed, addStatic)

	// An immovable wall weighs infinitely, so the impulse is A's alone:
	// -(1+1) * -4 / (1/DefaultMass) = 8, rather than the 4 an equal partner
	// would have taken half of.
	got := contactsBy(t, contactsQ, contactsComp)[idA]
	if len(got) != 1 {
		t.Fatalf("A recorded %d contacts, want 1", len(got))
	}
	if math.Abs(got[0].Impact-8) > 1e-9 {
		t.Errorf("impact against the wall = %v, want 8", got[0].Impact)
	}
	if p := findPos(t, qA, posA, idA); p.TopLeft.X == 100 {
		t.Error("expected dynamic A to have moved")
	}
	if p := findPos(t, qB, posB, idB); p.TopLeft.X != 105 {
		t.Errorf("expected static B to stay put at X=105, got X=%v", p.TopLeft.X)
	}
}

func TestNarrowPhase_SensorContact_RecordedButNeverPushed(t *testing.T) {
	space := testSpace(t)
	var posA goke.Comp[world.Position]
	var idA uid.UID64
	var qA *goke.Query
	var sensorEditor *goke.Editor

	var idB uid.UID64
	var contactsComp goke.Comp[collisions.Contacts]
	var contactsQ *goke.Query

	seed := goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var velA goke.Comp[world.Velocity]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &collA, &contactsComp)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10) // plenty of margin from the world edge
		velA.Slice(&fa.Cursor)[0].SetDelta(geom.NewVec(5, 0))
		idA = fa.IDs[0]

		var posB goke.Comp[world.Position]
		var velB goke.Comp[world.Velocity]
		fb := si.NewFactory(&posB, &velB)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10)
		idB = fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		qA = si.NewQueryBuilder(&posA).Include(goke.Include[world.Velocity]()).Build()
		contactsQ = si.NewQueryBuilder(&contactsComp).Build()
	}}

	// Sensor is a zero-size tag: Factory/Track reject zero-size data columns,
	// so it has to be added via an Editor migration (comp.Add has no such
	// restriction), in a second Setup-phase system run after the seed.
	var sensorComp goke.Comp[collisions.Sensor]
	addSensor := goke.SystemFn{
		OnInit: func(si *goke.SysInit) {
			sensorEditor = qA.NewEditorBuilder(&sensorComp).Build()
		},
		OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
			qA.All()
			for qA.Next() {
				buf := qA.BeginMigrate(cb)
				for _, id := range qA.Cursor().IDs {
					buf.Add(id)
				}
				buf.Commit(sensorEditor)
			}
		},
	}

	runNarrowPhase(t, space, seed, addSensor)

	if p := findPos(t, qA, posA, idA); p.TopLeft.X != 100 {
		t.Errorf("expected a sensor contact to never physically push A, TopLeft.X = %v, want 100", p.TopLeft.X)
	}

	// Being told is the whole point of a sensor, so the contact is still
	// published — with no impulse, since none was exchanged.
	got := contactsBy(t, contactsQ, contactsComp)[idA]
	if len(got) != 1 {
		t.Fatalf("sensor A recorded %d contacts, want 1", len(got))
	}
	if got[0].Other != idB || got[0].Impact != 0 {
		t.Errorf("sensor A recorded %+v, want a contact with %v at zero impact", got[0], idB)
	}
}

// The solver stops as soon as a whole pass separates nothing, which is safe
// only because a pass that moved nothing leaves the geometry it just judged
// untouched. A stack of three boxes is where that reasoning earns its keep:
// pushing the first pair apart drives the middle box into the third, so the
// second pass has work the first could not have seen. Stop too eagerly and
// the outer boxes stay inside each other.
func TestNarrowPhase_KeepsIteratingWhileSeparationCreatesNewOverlap(t *testing.T) {
	space := testSpace(t)
	var idA, idB, idC uid.UID64
	var posComp goke.Comp[world.Position]
	var q *goke.Query

	runNarrowPhase(t, space, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[world.Position]
		var vel goke.Comp[world.Velocity]
		var coll goke.Comp[collisions.Collision]

		f := si.NewFactory(&pos, &vel, &coll)
		f.Create(3)
		f.Next()
		slice := pos.Slice(&f.Cursor)
		// Three 10-wide boxes overlapping 8 units each: a tight stack, so
		// separating any pair pushes into the next.
		slice[0] = posAt(100, 100, 10, 10)
		slice[1] = posAt(102, 100, 10, 10)
		slice[2] = posAt(104, 100, 10, 10)
		idA, idB, idC = f.IDs[0], f.IDs[1], f.IDs[2]

		touching := coll.Slice(&f.Cursor)
		for i := range touching {
			for j, id := range f.IDs {
				if i != j {
					touching[i].Touching[touching[i].TouchingCount] = id
					touching[i].TouchingCount++
				}
			}
		}
		for i, id := range f.IDs {
			space.Insert(id, slice[i].AABB)
		}
		space.Flush(nil)

		posComp = pos
		q = si.NewQueryBuilder(&pos).Build()
	}})

	a := findPos(t, q, posComp, idA)
	b := findPos(t, q, posComp, idB)
	c := findPos(t, q, posComp, idC)

	for _, pair := range []struct {
		name string
		l, r world.Position
	}{{"A/B", a, b}, {"B/C", b, c}, {"A/C", a, c}} {
		// Resting exactly edge-to-edge is the right answer, so what is
		// measured is depth, not whether the closed boxes touch.
		if d := overlapDepth(pair.l, pair.r); d > 1e-6 {
			t.Errorf("%s still overlap by %v after the solver ran: %v vs %v", pair.name, d, pair.l.AABB.AABB, pair.r.AABB.AABB)
		}
	}
}

// A tick's contacts are settled in turn, not side by side. An entity squeezed
// between two others has to come out of the tick moving, exactly as it would
// have if each pair had been settled on its own; measuring both contacts
// against the same starting velocities instead leaves their impulses
// cancelling, and a row like this one stops dead — which is how clumps form.
func TestNarrowPhase_SqueezedEntity_ImpulsesChainRatherThanCancel(t *testing.T) {
	space := testSpace(t)
	var ids [3]uid.UID64
	var contactsComp goke.Comp[collisions.Contacts]
	var q *goke.Query

	runNarrowPhase(t, space, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[world.Position]
		var vel goke.Comp[world.Velocity]
		var coll goke.Comp[collisions.Collision]
		f := si.NewFactory(&pos, &vel, &coll, &contactsComp)
		f.Create(3)
		f.Next()
		positions := pos.Slice(&f.Cursor)
		velocities := vel.Slice(&f.Cursor)

		// A row of 10-wide boxes overlapping 3 units each, the outer two
		// closing on the middle one, which stands still.
		for i, x := range []float64{93, 100, 107} {
			positions[i] = posAt(x, 100, 10, 10)
			ids[i] = f.IDs[i]
			space.Insert(f.IDs[i], positions[i].AABB)
		}
		velocities[0].SetDelta(geom.NewVec(5, 0))
		velocities[2].SetDelta(geom.NewVec(-5, 0))
		space.Flush(nil)

		touching := coll.Slice(&f.Cursor)
		touching[0].Touching[0], touching[0].TouchingCount = f.IDs[1], 1
		touching[1].Touching[0], touching[1].Touching[1], touching[1].TouchingCount = f.IDs[0], f.IDs[2], 2
		touching[2].Touching[0], touching[2].TouchingCount = f.IDs[1], 1

		q = si.NewQueryBuilder(&contactsComp).Build()
	}})

	byEntity := contactsBy(t, q, contactsComp)
	middle := byEntity[ids[1]]
	if len(middle) != 2 {
		t.Fatalf("the middle entity recorded %d contacts, want 2", len(middle))
	}
	// Settling the first contact costs 5; the second is measured against what
	// that one left behind, so it costs 10 rather than another 5.
	for i, want := range []struct {
		other  uid.UID64
		impact float64
	}{{ids[0], 5}, {ids[2], 10}} {
		if middle[i].Other != want.other || math.Abs(middle[i].Impact-want.impact) > 1e-9 {
			t.Errorf("contact %d = (%v, impact %v), want (%v, %v)", i, middle[i].Other, middle[i].Impact, want.other, want.impact)
		}
	}

	// Which is the row swapping velocities down the line, rather than everyone
	// stopping where they stand.
	for i, start := range []float64{5, 0, -5} {
		got := start
		for _, c := range byEntity[ids[i]] {
			got += c.Impact * c.Normal.X // every entity here weighs DefaultMass
		}
		if want := []float64{0, -5, 5}[i]; math.Abs(got-want) > 1e-9 {
			t.Errorf("entity %d leaves the tick at %v, want %v", i, got, want)
		}
	}
}

// Chaining is about a tick's later contacts, so a pair on its own must settle
// for exactly what the two sides brought to it.
func TestNarrowPhase_LonePair_SettlesOnItsOwnVelocities(t *testing.T) {
	space := testSpace(t)
	var idA uid.UID64
	var contactsComp goke.Comp[collisions.Contacts]
	var q *goke.Query

	runNarrowPhase(t, space, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[world.Position]
		var vel goke.Comp[world.Velocity]
		var coll goke.Comp[collisions.Collision]
		f := si.NewFactory(&pos, &vel, &coll, &contactsComp)
		f.Create(2)
		f.Next()
		positions := pos.Slice(&f.Cursor)
		velocities := vel.Slice(&f.Cursor)
		positions[0] = posAt(100, 100, 10, 10)
		positions[1] = posAt(107, 100, 10, 10)
		velocities[0].SetDelta(geom.NewVec(5, 0))
		velocities[1].SetDelta(geom.NewVec(-5, 0))
		idA = f.IDs[0]

		touching := coll.Slice(&f.Cursor)
		touching[0].Touching[0], touching[0].TouchingCount = f.IDs[1], 1

		q = si.NewQueryBuilder(&contactsComp).Build()
	}})

	// -(1+1) * (5 - -5) / (1 + 1), closing head-on at equal weight.
	got := contactsBy(t, q, contactsComp)[idA]
	if len(got) != 1 {
		t.Fatalf("recorded %d contacts, want 1", len(got))
	}
	if math.Abs(got[0].Impact-10) > 1e-9 {
		t.Errorf("impact = %v, want 10", got[0].Impact)
	}
}

// Each side's material comes from its own components, and only a side that
// carries none falls back: A is heavy and barely springy, B carries a
// meaningless Mass and no Restitution at all. Both show up in the one number
// the pair publishes.
func TestNarrowPhase_Material_ReadPerSideWithFallbacks(t *testing.T) {
	space := testSpace(t)
	var idA uid.UID64
	var contactsComp goke.Comp[collisions.Contacts]
	var q *goke.Query

	runNarrowPhase(t, space, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posA goke.Comp[world.Position]
		var velA goke.Comp[world.Velocity]
		var collA goke.Comp[collisions.Collision]
		var massA goke.Comp[collisions.Mass]
		var restitutionA goke.Comp[collisions.Restitution]
		fa := si.NewFactory(&posA, &velA, &collA, &massA, &restitutionA, &contactsComp)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10)
		velA.Slice(&fa.Cursor)[0].SetDelta(geom.NewVec(5, 0)) // closing on B
		massA.Slice(&fa.Cursor)[0] = collisions.Mass{Value: 4}
		restitutionA.Slice(&fa.Cursor)[0] = collisions.Restitution{Value: 0.5}
		idA = fa.IDs[0]

		var posB goke.Comp[world.Position]
		var velB goke.Comp[world.Velocity]
		var massB goke.Comp[collisions.Mass]
		fb := si.NewFactory(&posB, &velB, &massB)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10)
		velB.Slice(&fb.Cursor)[0].SetDelta(geom.NewVec(-5, 0))
		massB.Slice(&fb.Cursor)[0] = collisions.Mass{Value: 0} // meaningless -> DefaultMass

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = fb.IDs[0]
		coll[0].TouchingCount = 1

		q = si.NewQueryBuilder(&contactsComp).Build()
	}})

	// -(1 + min(0.5, 1)) * -10 / (1/4 + 1/DefaultMass) = 12, where reading
	// either side's material as a default would have given 10.
	got := contactsBy(t, q, contactsComp)[idA]
	if len(got) != 1 {
		t.Fatalf("A recorded %d contacts, want 1", len(got))
	}
	if math.Abs(got[0].Impact-12) > 1e-9 {
		t.Errorf("impact = %v, want 12 (mass 4 and restitution 0.5 on A, defaults on B)", got[0].Impact)
	}
}

// A confirmed contact is published to both sides that asked for it, naming the
// other entity and how hard the two met — this is what game logic reads a tick
// later instead of digging through the solver.
func TestNarrowPhase_Contacts_PublishedToBothSides(t *testing.T) {
	space := testSpace(t)
	var idA, idB uid.UID64
	var contactsComp goke.Comp[collisions.Contacts]
	var q *goke.Query

	runNarrowPhase(t, space, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posA goke.Comp[world.Position]
		var velA goke.Comp[world.Velocity]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &collA, &contactsComp)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10)
		velA.Slice(&fa.Cursor)[0].SetDelta(geom.NewVec(5, 0)) // closing on B
		idA = fa.IDs[0]

		var posB goke.Comp[world.Position]
		var velB goke.Comp[world.Velocity]
		var contactsB goke.Comp[collisions.Contacts]
		fb := si.NewFactory(&posB, &velB, &contactsB)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10) // overlaps A by 5px on X
		velB.Slice(&fb.Cursor)[0].SetDelta(geom.NewVec(-5, 0))
		idB = fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		q = si.NewQueryBuilder(&contactsComp).Build()
	}})

	byEntity := contactsBy(t, q, contactsComp)
	for _, side := range []struct {
		self, other uid.UID64
	}{{idA, idB}, {idB, idA}} {
		got := byEntity[side.self]
		if len(got) != 1 {
			t.Fatalf("entity %v recorded %d contacts, want 1", side.self, len(got))
		}
		if got[0].Other != side.other {
			t.Errorf("entity %v recorded a contact with %v, want %v", side.self, got[0].Other, side.other)
		}
		if got[0].Impact <= 0 {
			t.Errorf("entity %v recorded impact %v, want the closing speed to register", side.self, got[0].Impact)
		}
	}
	if byEntity[idA][0].Impact != byEntity[idB][0].Impact {
		t.Errorf("sides disagree on the impulse: %v vs %v", byEntity[idA][0].Impact, byEntity[idB][0].Impact)
	}
}

// Nothing may linger: a contact belongs to the tick it happened in, or the wolf
// eats the same hare again next tick.
func TestNarrowPhase_Contacts_DoNotSurviveTheNextTick(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var contactsComp goke.Comp[collisions.Contacts]
	var q *goke.Query

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		seedContactReporter(t, si, space, posAt(100, 100, 10, 10), geom.NewVec(5, 0), &contactsComp)
		var other goke.Comp[collisions.Contacts]
		seedContactReporter(t, si, space, posAt(105, 100, 10, 10), geom.NewVec(-5, 0), &other)
		space.Flush(nil)
		q = si.NewQueryBuilder(&contactsComp).Build()
	}})

	broad := ecs.RegSys(collisions.NewBroadPhase(space, testProbeMargin))
	narrow := ecs.RegSys(collisions.NewNarrowPhase(space))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(broad, d)
		ctx.Sync()
		ctx.Run(narrow, d)
		ctx.Sync()
	})

	ecs.Tick(time.Millisecond)
	if got := countContacts(t, q, contactsComp); got != 2 {
		t.Fatalf("%d contacts recorded across both entities on the overlapping tick, want 2", got)
	}

	// The solver pushed them apart, so this tick confirms nothing — and last
	// tick's contacts must be gone rather than read a second time.
	ecs.Tick(time.Millisecond)
	if got := countContacts(t, q, contactsComp); got != 0 {
		t.Errorf("%d contacts left after a tick with no contact, want 0", got)
	}
}

// seedContactReporter spawns a collidable entity that records what it strikes, moving at delta.
func seedContactReporter(t *testing.T, si *goke.SysInit, space *gokg.Space, pos world.Position, delta geom.Vec, contacts *goke.Comp[collisions.Contacts]) uid.UID64 {
	t.Helper()
	var posComp goke.Comp[world.Position]
	var velComp goke.Comp[world.Velocity]
	var collComp goke.Comp[collisions.Collision]
	f := si.NewFactory(&posComp, &velComp, &collComp, contacts)
	f.Create(1)
	f.Next()
	posComp.Slice(&f.Cursor)[0] = pos
	velComp.Slice(&f.Cursor)[0].SetDelta(delta)
	id := f.IDs[0]
	space.Insert(id, pos.AABB)
	space.SetCapabilities(id, collisions.CanCollide)
	return id
}

// contactsBy collects what each entity q matches recorded this tick.
func contactsBy(t *testing.T, q *goke.Query, comp goke.Comp[collisions.Contacts]) map[uid.UID64][]collisions.Contact {
	t.Helper()
	found := map[uid.UID64][]collisions.Contact{}
	q.All()
	for q.Next() {
		cur := q.Cursor()
		slice := comp.Slice(cur)
		for i, id := range cur.IDs {
			found[id] = append(found[id], slice[i].All()...)
		}
	}
	return found
}

func countContacts(t *testing.T, q *goke.Query, comp goke.Comp[collisions.Contacts]) int {
	t.Helper()
	var n int
	for _, contacts := range contactsBy(t, q, comp) {
		n += len(contacts)
	}
	return n
}

// overlapDepth is how far two boxes penetrate on their shallower axis, and
// zero (or less) when they only touch or stand apart.
func overlapDepth(l, r world.Position) float64 {
	x := min(l.BottomRight.X, r.BottomRight.X) - max(l.TopLeft.X, r.TopLeft.X)
	y := min(l.BottomRight.Y, r.BottomRight.Y) - max(l.TopLeft.Y, r.TopLeft.Y)
	return min(x, y)
}
