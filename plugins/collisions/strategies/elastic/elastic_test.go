package elastic_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/elastic"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/uid"
)

const epsilon = 1e-9

// entity is one side of a contact as the narrow phase left it: a velocity, a
// weight, and what it struck.
type entity struct {
	delta   geom.Vec
	mass    float64
	bouncy  bool
	contact collisions.Contact
}

// run ticks the behavior over entities and returns their velocities after it.
func run(t *testing.T, entities ...entity) []geom.Vec {
	t.Helper()
	var velComp goke.Comp[world.Velocity]
	var contactsComp goke.Comp[collisions.Contacts]
	var massComp goke.Comp[collisions.Mass]
	var bouncyComp goke.Comp[elastic.Bouncy]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		for _, e := range entities {
			var f *goke.Factory
			if e.bouncy {
				f = si.NewFactory(&velComp, &contactsComp, &massComp, &bouncyComp)
			} else {
				f = si.NewFactory(&velComp, &contactsComp, &massComp)
			}
			f.Create(1)
			f.Next()
			velComp.Slice(&f.Cursor)[0].SetDelta(e.delta)
			massComp.Slice(&f.Cursor)[0] = collisions.Mass{Value: e.mass}
			var c collisions.Contacts
			c.Items[0] = e.contact
			c.Count = 1
			contactsComp.Slice(&f.Cursor)[0] = c
		}
		q = si.NewQueryBuilder(&velComp).Build()
	}})

	handle := ecs.RegSys(elastic.New())
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	var got []geom.Vec
	q.All()
	for q.Next() {
		for _, v := range velComp.Slice(q.Cursor()) {
			got = append(got, v.Delta())
		}
	}
	return got
}

func struck(impact float64, normal geom.Vec) collisions.Contact {
	return collisions.Contact{Other: uid.UID64(9), Impact: impact, Normal: normal}
}

// Two equally heavy sides, each reading its own normal, end up having swapped
// velocities — the classic head-on bounce, now assembled from two independent
// halves.
func TestBehavior_EqualMasses_ExchangeVelocities(t *testing.T) {
	const impact = 7 // -(1+1) * (-3-4) / (1+1)

	got := run(t,
		entity{delta: geom.NewVec(-3, 2), mass: 1, bouncy: true, contact: struck(impact, geom.NewVec(1, 0))},
		entity{delta: geom.NewVec(4, 2), mass: 1, bouncy: true, contact: struck(impact, geom.NewVec(-1, 0))},
	)

	if len(got) != 2 {
		t.Fatalf("got %d velocities, want 2", len(got))
	}
	if math.Abs(got[0].X-4) > epsilon || math.Abs(got[1].X+3) > epsilon {
		t.Errorf("X = (%v, %v), want (4, -3) swapped", got[0].X, got[1].X)
	}
	if math.Abs(got[0].Y-2) > epsilon || math.Abs(got[1].Y-2) > epsilon {
		t.Errorf("Y = (%v, %v), want both left at 2 — nothing acts across the normal", got[0].Y, got[1].Y)
	}
}

// The same impulse moves a heavy side less, in inverse proportion to weight.
func TestBehavior_ShareIsInverseToWeight(t *testing.T) {
	const impact = 7.2

	got := run(t,
		entity{delta: geom.NewVec(2, 0), mass: 9, bouncy: true, contact: struck(impact, geom.NewVec(-1, 0))},
		entity{delta: geom.NewVec(-2, 0), mass: 1, bouncy: true, contact: struck(impact, geom.NewVec(1, 0))},
	)

	if math.Abs(got[0].X-1.2) > epsilon {
		t.Errorf("heavy side = %v, want 1.2 (2 - 7.2/9)", got[0].X)
	}
	if math.Abs(got[1].X-5.2) > epsilon {
		t.Errorf("light side = %v, want 5.2 (-2 + 7.2)", got[1].X)
	}
}

func TestBehavior_WithoutBouncy_IsLeftAlone(t *testing.T) {
	got := run(t, entity{delta: geom.NewVec(-3, 0), mass: 1, contact: struck(7, geom.NewVec(1, 0))})

	if len(got) != 1 || math.Abs(got[0].X+3) > epsilon {
		t.Errorf("velocity = %v, want the entity untouched without Bouncy", got)
	}
}

// A contact the narrow phase measured as nothing — they were drawing apart, or
// it was only sensed — must move no one.
func TestBehavior_ZeroImpact_ChangesNothing(t *testing.T) {
	got := run(t, entity{delta: geom.NewVec(3, 0), mass: 1, bouncy: true, contact: struck(0, geom.NewVec(1, 0))})

	if len(got) != 1 || math.Abs(got[0].X-3) > epsilon {
		t.Errorf("velocity = %v, want it untouched", got)
	}
}
