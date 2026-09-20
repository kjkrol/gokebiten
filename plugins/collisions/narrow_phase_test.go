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

const epsilon = 1e-9

// thing is one entity of a narrow-phase fixture — a 10x10 box at y=100 — and,
// once the tick has run, what became of it.
type thing struct {
	x        float64
	delta    geom.Vec
	physics  *collisions.Physics // nil: only ever detected
	touching []int               // fixture indices it lists as broad-phase candidates

	id       uid.UID64
	base     world.Base
	contacts []collisions.Contact
}

func elastic(mass float64) *collisions.Physics {
	return &collisions.Physics{Mass: mass, Restitution: 1}
}

// narrowTick spawns things in order — so a lower fixture index is a lower
// entity index — runs one narrow-phase tick over them, and reads each back.
func narrowTick(t *testing.T, things ...*thing) {
	t.Helper()
	space := testSpace(t)

	var base goke.Comp[world.Base]
	var coll goke.Comp[collisions.Collision]
	var physics goke.Comp[collisions.Physics]
	var seen goke.Comp[collisions.Collision]
	var read *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		for _, th := range things {
			comps := []goke.Addable{&base, &coll}
			if th.physics != nil {
				comps = append(comps, &physics)
			}
			f := si.NewFactory(comps...)
			f.Create(1)
			f.Next()
			th.id = f.IDs[0]
			placed := world.Base{Pos: posAt(th.x, 100, 10, 10)}
			placed.Vel.SetDelta(th.delta)
			base.Slice(&f.Cursor)[0] = placed
			if th.physics != nil {
				physics.Slice(&f.Cursor)[0] = *th.physics
			}
			space.Insert(th.id, placed.Pos.AABB)
		}
		space.Flush(nil)

		// Candidates are named by fixture index, and ids exist only now.
		byID := map[uid.UID64]*thing{}
		for _, th := range things {
			byID[th.id] = th
		}
		all := si.NewQueryBuilder(&coll).Build()
		for all.All(); all.Next(); {
			cursor := all.Cursor()
			lists := coll.Slice(cursor)
			for i, id := range cursor.IDs {
				for _, other := range byID[id].touching {
					lists[i].Touching[lists[i].TouchingCount] = things[other].id
					lists[i].TouchingCount++
				}
			}
		}
		read = si.NewQueryBuilder(&base, &seen).Build()
	}})

	handle := ecs.RegSys(collisions.NewNarrowPhase(space))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	byID := map[uid.UID64]*thing{}
	for _, th := range things {
		byID[th.id] = th
	}
	for read.All(); read.Next(); {
		cursor := read.Cursor()
		bases := base.Slice(cursor)
		recorded := seen.Slice(cursor)
		for i, id := range cursor.IDs {
			th := byID[id]
			th.base = bases[i]
			th.contacts = append([]collisions.Contact(nil), recorded[i].Contacts()...)
		}
	}
}

func (th *thing) left() float64 { return float64(th.base.Pos.TopLeft.X) }

func (th *thing) speedX() float64 { return th.base.Vel.Delta().X }

func TestNarrowPhase_PhysicalPair_IsPushedApart(t *testing.T) {
	a := &thing{x: 100, physics: elastic(1), touching: []int{1}}
	b := &thing{x: 105, physics: elastic(1)}

	narrowTick(t, a, b)

	if a.left() >= 100 || b.left() <= 105 {
		t.Errorf("a at %v, b at %v — want both pushed out of a 5-unit overlap", a.left(), b.left())
	}
}

// An infinite mass is what a wall is: it holds its ground, and whatever runs
// into it is pushed all the way out and sent back the way it came.
func TestNarrowPhase_ImmovableSide_StaysPutAndReflectsTheOther(t *testing.T) {
	ball := &thing{x: 100, delta: geom.NewVec(4, 0), physics: elastic(1), touching: []int{1}}
	wall := &thing{x: 105, physics: elastic(math.Inf(1))}

	narrowTick(t, ball, wall)

	if wall.left() != 105 {
		t.Errorf("the wall moved to %v, want it left at 105", wall.left())
	}
	if ball.left() != 95 {
		t.Errorf("the ball is at %v, want it pushed the whole 5 units out, to 95", ball.left())
	}
	if math.Abs(ball.speedX()+4) > epsilon {
		t.Errorf("the ball moves at %v, want -4 — straight back off the wall", ball.speedX())
	}
	// The impulse is the ball's alone: -(1+1) * -4 / (1/DefaultMass) = 8, not
	// the 4 an equal partner would have taken half of.
	if len(ball.contacts) != 1 || math.Abs(ball.contacts[0].Impact-8) > epsilon {
		t.Errorf("contacts = %+v, want one at impact 8", ball.contacts)
	}
}

// A pair is built from whichever side has the lower index, so an immovable
// entity spawned before the one that runs into it has to be named static on
// its own side of the pair — or the solver shoves the wall.
func TestNarrowPhase_ImmovableSpawnedFirst_StillStopsWhatRunsIntoIt(t *testing.T) {
	wall := &thing{x: 105, physics: elastic(math.Inf(1)), touching: []int{1}}
	ball := &thing{x: 100, delta: geom.NewVec(4, 0), physics: elastic(1), touching: []int{0}}

	narrowTick(t, wall, ball)

	if wall.id.Index() >= ball.id.Index() {
		t.Fatalf("fixture broken: wall index %d, ball index %d — the wall has to come first", wall.id.Index(), ball.id.Index())
	}
	if wall.left() != 105 {
		t.Errorf("the wall moved to %v, want it left at 105", wall.left())
	}
	if ball.left() != 95 || math.Abs(ball.speedX()+4) > epsilon {
		t.Errorf("the ball is at %v moving %v, want 95 and -4", ball.left(), ball.speedX())
	}
}

// An entity with no Physics takes no part in the physical world: a town to walk
// into, a trigger, a hunter's jaws. The contact is still reported — that is the
// point of it — but nothing is pushed and no impulse changes hands.
func TestNarrowPhase_SideWithoutPhysics_IsDetectedButNeverPushed(t *testing.T) {
	for name, town := range map[string]*thing{
		"spawned after the walker":  {x: 105},
		"spawned before the walker": {x: 105, touching: []int{1}},
	} {
		t.Run(name, func(t *testing.T) {
			walker := &thing{x: 100, delta: geom.NewVec(4, 0), physics: elastic(1), touching: []int{1}}
			order := []*thing{walker, town}
			if len(town.touching) > 0 {
				walker.touching = []int{0}
				order = []*thing{town, walker}
			}

			narrowTick(t, order...)

			if walker.left() != 100 || town.left() != 105 {
				t.Errorf("walker at %v, town at %v — want neither pushed", walker.left(), town.left())
			}
			if math.Abs(walker.speedX()-4) > epsilon {
				t.Errorf("the walker moves at %v, want its 4 untouched", walker.speedX())
			}
			for who, th := range map[string]*thing{"walker": walker, "town": town} {
				if len(th.contacts) != 1 || th.contacts[0].Impact != 0 {
					t.Errorf("%s contacts = %+v, want exactly one, at zero impact", who, th.contacts)
				}
			}
			if len(walker.contacts) == 1 && walker.contacts[0].Other != town.id {
				t.Errorf("the walker struck %v, want the town %v", walker.contacts[0].Other, town.id)
			}
		})
	}
}

// The solver stops as soon as a whole pass separates nothing, which is safe
// only because a pass that moved nothing leaves the geometry it just judged
// untouched. A stack of three boxes is where that reasoning earns its keep:
// pushing the first pair apart drives the middle box into the third, so the
// second pass has work the first could not have seen. Stop too eagerly and
// the outer boxes stay inside each other.
func TestNarrowPhase_KeepsIteratingWhileSeparationCreatesNewOverlap(t *testing.T) {
	// Three 10-wide boxes overlapping 8 units each: a tight stack, so
	// separating any pair pushes into the next.
	a := &thing{x: 100, physics: elastic(1), touching: []int{1, 2}}
	b := &thing{x: 102, physics: elastic(1), touching: []int{0, 2}}
	c := &thing{x: 104, physics: elastic(1), touching: []int{0, 1}}

	narrowTick(t, a, b, c)

	for _, pair := range []struct {
		name string
		l, r *thing
	}{{"A/B", a, b}, {"B/C", b, c}, {"A/C", a, c}} {
		// Resting exactly edge-to-edge is the right answer, so what is
		// measured is depth, not whether the closed boxes touch.
		if d := overlapDepth(pair.l.base.Pos, pair.r.base.Pos); d > 1e-6 {
			t.Errorf("%s still overlap by %v after the solver ran", pair.name, d)
		}
	}
}

// A tick's contacts are settled in turn, each from what the one before left
// behind. An entity squeezed between two others has to come out of the tick
// moving; measured side by side against the same starting velocities, the two
// impulses cancel and the whole row stops dead — which is how clumps form.
func TestNarrowPhase_SqueezedEntity_BouncesOffBothNeighboursInTurn(t *testing.T) {
	left := &thing{x: 93, delta: geom.NewVec(5, 0), physics: elastic(1), touching: []int{1}}
	middle := &thing{x: 100, physics: elastic(1), touching: []int{0, 2}}
	right := &thing{x: 107, delta: geom.NewVec(-5, 0), physics: elastic(1), touching: []int{1}}

	narrowTick(t, left, middle, right)

	// The row swaps velocities down the line, exactly as settling each pair on
	// its own would: 5,0 -> 0,5 and then 5,-5 -> -5,5.
	for i, want := range []float64{0, -5, 5} {
		if got := []*thing{left, middle, right}[i].speedX(); math.Abs(got-want) > epsilon {
			t.Errorf("entity %d leaves the tick at %v, want %v", i, got, want)
		}
	}
	if len(middle.contacts) != 2 {
		t.Fatalf("the middle entity recorded %d contacts, want 2", len(middle.contacts))
	}
	// Settling the first contact costs 5; the second is measured against what
	// that one left behind, so it costs 10 rather than another 5.
	for i, want := range []struct {
		other  uid.UID64
		impact float64
	}{{left.id, 5}, {right.id, 10}} {
		if got := middle.contacts[i]; got.Other != want.other || math.Abs(got.Impact-want.impact) > epsilon {
			t.Errorf("contact %d = (%v, impact %v), want (%v, %v)", i, got.Other, got.Impact, want.other, want.impact)
		}
	}
}

// The classic head-on case: equal weights trade velocities along the contact
// axis and keep whatever they had across it.
func TestNarrowPhase_EqualMasses_ExchangeVelocitiesAlongTheNormal(t *testing.T) {
	a := &thing{x: 100, delta: geom.NewVec(5, 2), physics: elastic(1), touching: []int{1}}
	b := &thing{x: 107, delta: geom.NewVec(-5, 2), physics: elastic(1)}

	narrowTick(t, a, b)

	if math.Abs(a.speedX()+5) > epsilon || math.Abs(b.speedX()-5) > epsilon {
		t.Errorf("X = (%v, %v), want (-5, 5) swapped", a.speedX(), b.speedX())
	}
	if ay, by := a.base.Vel.Delta().Y, b.base.Vel.Delta().Y; math.Abs(ay-2) > epsilon || math.Abs(by-2) > epsilon {
		t.Errorf("Y = (%v, %v), want both left at 2 — nothing acts across the normal", ay, by)
	}
}

// The impulse is shared out by weight, so a heavy side barely deflects while
// the light one is thrown back — momentum and energy are what pin it down.
func TestNarrowPhase_HeavyAgainstLight_ConservesMomentumAndEnergy(t *testing.T) {
	const heavyMass, lightMass = 9, 1
	heavy := &thing{x: 100, delta: geom.NewVec(2, 0), physics: elastic(heavyMass), touching: []int{1}}
	light := &thing{x: 107, delta: geom.NewVec(-2, 0), physics: elastic(lightMass)}

	narrowTick(t, heavy, light)

	h, l := heavy.speedX(), light.speedX()
	if got, want := heavyMass*h+lightMass*l, float64(heavyMass*2+lightMass*-2); math.Abs(got-want) > epsilon {
		t.Errorf("momentum = %v, want %v", got, want)
	}
	if got, want := heavyMass*h*h+lightMass*l*l, float64(heavyMass*4+lightMass*4); math.Abs(got-want) > epsilon {
		t.Errorf("kinetic energy (x2) = %v, want %v", got, want)
	}
	if h <= 0 || l <= 0 {
		t.Errorf("heavy moves at %v, light at %v — want the heavy one carrying on and the light one thrown back", h, l)
	}
}

// A pair bounces by the softer of its two sides, all the way down to not at all.
func TestNarrowPhase_Restitution_DampsTheBounce(t *testing.T) {
	cases := map[string]struct {
		restitutionA, restitutionB float64
		wantA, wantB               float64
	}{
		"perfectly elastic":   {1, 1, 4, -3},
		"half, from A":        {0.5, 1, 2.25, -1.25},
		"half, from B":        {1, 0.5, 2.25, -1.25},
		"perfectly inelastic": {0, 1, 0.5, 0.5},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a := &thing{x: 107, delta: geom.NewVec(-3, 0), physics: &collisions.Physics{Restitution: c.restitutionA}, touching: []int{1}}
			b := &thing{x: 100, delta: geom.NewVec(4, 0), physics: &collisions.Physics{Restitution: c.restitutionB}}

			narrowTick(t, a, b)

			if math.Abs(a.speedX()-c.wantA) > epsilon || math.Abs(b.speedX()-c.wantB) > epsilon {
				t.Errorf("speeds = (%v, %v), want (%v, %v)", a.speedX(), b.speedX(), c.wantA, c.wantB)
			}
		})
	}
}

// Each side's material is its own: A is heavy and only half springy, B names no
// Mass at all and so weighs DefaultMass. Both show up in the one number the pair
// publishes — -(1 + min(0.5, 1)) * -10 / (1/4 + 1/DefaultMass) = 12, where
// reading either side's Physics wrongly would have given 10.
func TestNarrowPhase_Material_IsReadPerSide(t *testing.T) {
	a := &thing{x: 100, delta: geom.NewVec(5, 0), physics: &collisions.Physics{Mass: 4, Restitution: 0.5}, touching: []int{1}}
	b := &thing{x: 105, delta: geom.NewVec(-5, 0), physics: &collisions.Physics{Restitution: 1}}

	narrowTick(t, a, b)

	if len(a.contacts) != 1 || math.Abs(a.contacts[0].Impact-12) > epsilon {
		t.Errorf("contacts = %+v, want one at impact 12", a.contacts)
	}
}

// A confirmed contact is published to both sides that asked for it, naming the
// other entity, how hard the two met, and which way each of them left.
func TestNarrowPhase_Contacts_PublishedToBothSides(t *testing.T) {
	a := &thing{x: 100, delta: geom.NewVec(5, 0), physics: elastic(1), touching: []int{1}}
	b := &thing{x: 105, delta: geom.NewVec(-5, 0), physics: elastic(1)}

	narrowTick(t, a, b)

	for _, side := range []struct {
		self, other *thing
		leaves      float64
	}{{a, b, -1}, {b, a, 1}} {
		if len(side.self.contacts) != 1 {
			t.Fatalf("entity %v recorded %d contacts, want 1", side.self.id, len(side.self.contacts))
		}
		got := side.self.contacts[0]
		if got.Other != side.other.id {
			t.Errorf("entity %v recorded a contact with %v, want %v", side.self.id, got.Other, side.other.id)
		}
		if math.Abs(got.Impact-10) > epsilon {
			t.Errorf("entity %v recorded impact %v, want 10", side.self.id, got.Impact)
		}
		if got.Normal.X != side.leaves || got.Normal.Y != 0 {
			t.Errorf("entity %v was told to leave along %v, want (%v, 0) — away from the other", side.self.id, got.Normal, side.leaves)
		}
	}
}

// Nothing may linger: a contact belongs to the tick it happened in, or the wolf
// eats the same hare again next tick.
func TestNarrowPhase_Contacts_DoNotSurviveTheNextTick(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var struck goke.Comp[collisions.Collision]
	var q *goke.Query

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		seedPhysical(t, si, space, posAt(100, 100, 10, 10), geom.NewVec(5, 0))
		seedPhysical(t, si, space, posAt(105, 100, 10, 10), geom.NewVec(-5, 0))
		space.Flush(nil)
		q = si.NewQueryBuilder(&struck).Build()
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
	if got := countContacts(q, struck); got != 2 {
		t.Fatalf("%d contacts recorded across both entities on the overlapping tick, want 2", got)
	}

	// The solver pushed them apart, so this tick confirms nothing — and last
	// tick's contacts must be gone rather than read a second time.
	ecs.Tick(time.Millisecond)
	if got := countContacts(q, struck); got != 0 {
		t.Errorf("%d contacts left after a tick with no contact, want 0", got)
	}
}

// seedPhysical spawns a physical, collidable entity moving at delta.
func seedPhysical(t *testing.T, si *goke.SysInit, space *gokg.Space, pos world.Position, delta geom.Vec) uid.UID64 {
	t.Helper()
	var baseComp goke.Comp[world.Base]
	var collComp goke.Comp[collisions.Collision]
	var physicsComp goke.Comp[collisions.Physics]
	f := si.NewFactory(&baseComp, &collComp, &physicsComp)
	f.Create(1)
	f.Next()
	baseComp.Slice(&f.Cursor)[0].Pos = pos
	baseComp.Slice(&f.Cursor)[0].Vel.SetDelta(delta)
	physicsComp.Slice(&f.Cursor)[0] = collisions.Physics{Restitution: 1}
	id := f.IDs[0]
	space.Insert(id, pos.AABB)
	space.SetCapabilities(id, collisions.CanCollide)
	return id
}

func countContacts(q *goke.Query, comp goke.Comp[collisions.Collision]) int {
	var n int
	for q.All(); q.Next(); {
		for _, c := range comp.Slice(q.Cursor()) {
			n += len(c.Contacts())
		}
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
