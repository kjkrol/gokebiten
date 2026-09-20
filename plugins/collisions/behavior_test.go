package collisions_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

type bullet struct{ Damage int }

type target struct{ HP int }

// tagged is one box of a behavior fixture: where it is and which tags it carries.
type tagged struct {
	x              float64
	bullet, target bool
	id             uid.UID64
}

// registrar is the part of the collision engine a fixture registers behaviors on.
type registrar interface {
	RegisterBehavior(behaviors ...plugin.Behavior) error
}

// meet runs the real collision engine for one tick over boxes, with behaviors
// hosted in it, and returns every Meeting they were handed.
func meet(t *testing.T, behaviorsOf func(record func(collisions.Meeting)) []plugin.Behavior, boxes ...*tagged) []collisions.Meeting {
	t.Helper()
	var met []collisions.Meeting
	meetWith(t, func(engine registrar) {
		if err := engine.RegisterBehavior(behaviorsOf(func(m collisions.Meeting) { met = append(met, m) })...); err != nil {
			t.Fatalf("RegisterBehavior: %v", err)
		}
	}, boxes...)
	return met
}

// meetWith is meet with the registering left to the caller.
func meetWith(t *testing.T, register func(engine registrar), boxes ...*tagged) {
	t.Helper()
	space := testSpace(t)
	ecs := goke.New()
	engine := collisions.New(space, ecs, testProbeMargin)
	register(engine)

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var base goke.Comp[world.Base]
		var coll goke.Comp[collisions.Collision]
		var bullets goke.Comp[bullet]
		var targets goke.Comp[target]
		for _, box := range boxes {
			comps := []goke.Addable{&base, &coll}
			if box.bullet {
				comps = append(comps, &bullets)
			}
			if box.target {
				comps = append(comps, &targets)
			}
			f := si.NewFactory(comps...)
			f.Create(1)
			f.Next()
			box.id = f.IDs[0]
			placed := posAt(box.x, 100, 10, 10)
			base.Slice(&f.Cursor)[0].Pos = placed
			space.Insert(box.id, placed.AABB)
			space.SetCapabilities(box.id, collisions.CanCollide)
		}
		space.Flush(nil)
	}})
	engine.RegSystems(ecs)
	ecs.SetPlan(engine.RunPlan)
	ecs.Tick(time.Millisecond)
}

func bulletsAgainstTargets(record func(collisions.Meeting)) []plugin.Behavior {
	return []plugin.Behavior{plugin.Between[bullet, target](func(_ plugin.Tick, m collisions.Meeting) { record(m) })}
}

// Self is always the side carrying the first tag, whichever of the two was
// spawned first — and so whichever side of the pair the narrow phase walked.
func TestBetween_HandsOverThePairWithSelfOnTheFirstTag(t *testing.T) {
	for name, order := range map[string][2]bool{"bullet spawned first": {true, false}, "target spawned first": {false, true}} {
		t.Run(name, func(t *testing.T) {
			first := &tagged{x: 100, bullet: order[0], target: !order[0]}
			second := &tagged{x: 105, bullet: order[1], target: !order[1]}

			met := meet(t, bulletsAgainstTargets, first, second)

			shot, struck := first, second
			if !first.bullet {
				shot, struck = second, first
			}
			if len(met) != 1 {
				t.Fatalf("behavior ran %d times, want once", len(met))
			}
			if met[0].Self != shot.id || met[0].Other != struck.id {
				t.Errorf("Meeting = (self %v, other %v), want (bullet %v, target %v)", met[0].Self, met[0].Other, shot.id, struck.id)
			}
		})
	}
}

func TestBetween_IgnoresPairsThatDoNotCarryBothTags(t *testing.T) {
	met := meet(t, bulletsAgainstTargets,
		&tagged{x: 100, bullet: true}, &tagged{x: 105, bullet: true}, // two bullets
		&tagged{x: 300}, &tagged{x: 305, target: true}, // a bystander and a target
	)

	if len(met) != 0 {
		t.Errorf("behavior ran for %+v, want it left alone — no bullet met a target", met)
	}
}

// A pair of the same tag would match either way round, and is still one contact.
func TestBetween_SameTagOnBothSides_RunsOncePerContact(t *testing.T) {
	met := meet(t, func(record func(collisions.Meeting)) []plugin.Behavior {
		return []plugin.Behavior{plugin.Between[bullet, bullet](func(_ plugin.Tick, m collisions.Meeting) { record(m) })}
	}, &tagged{x: 100, bullet: true}, &tagged{x: 105, bullet: true})

	if len(met) != 1 {
		t.Errorf("behavior ran %d times, want once for one contact", len(met))
	}
}

// Anything stands for whatever is on the other side — tagged or not.
func TestBetween_Anything_MatchesWhateverIsThere(t *testing.T) {
	shot, wall := &tagged{x: 100, bullet: true}, &tagged{x: 105}

	met := meet(t, func(record func(collisions.Meeting)) []plugin.Behavior {
		return []plugin.Behavior{plugin.Between[bullet, plugin.Anything](func(_ plugin.Tick, m collisions.Meeting) { record(m) })}
	}, shot, wall)

	if len(met) != 1 || met[0].Self != shot.id || met[0].Other != wall.id {
		t.Errorf("Meetings = %+v, want the bullet %v meeting the untagged %v once", met, shot.id, wall.id)
	}
}

// Two behaviors naming the same tag share one optional component in the query —
// a query refuses to carry the same component twice.
func TestBetween_BehaviorsSharingATag_BothRun(t *testing.T) {
	var first, second int
	met := meet(t, func(func(collisions.Meeting)) []plugin.Behavior {
		return []plugin.Behavior{
			plugin.Between[bullet, target](func(plugin.Tick, collisions.Meeting) { first++ }),
			plugin.Between[bullet, plugin.Anything](func(plugin.Tick, collisions.Meeting) { second++ }),
		}
	}, &tagged{x: 100, bullet: true}, &tagged{x: 105, target: true})

	if first != 1 || second != 1 || len(met) != 0 {
		t.Errorf("behaviors ran (%d, %d) times, want (1, 1)", first, second)
	}
}

// A behavior says whose it is by its payload: collisions hosts pairs described
// as a Meeting and entities described as Struck, and nothing else.
func TestRegisterBehavior_RefusesWhatItCannotHost(t *testing.T) {
	engine := collisions.New(testSpace(t), goke.New(), testProbeMargin)

	for name, b := range map[string]plugin.Behavior{
		"not a behavior at all":          "just a string",
		"a pair made for another host":   plugin.Between[bullet, target](func(plugin.Tick, string) {}),
		"an entity made for another one": plugin.Each(func(plugin.Tick, *bullet, string) {}),
	} {
		if err := engine.RegisterBehavior(b); !errors.Is(err, plugin.ErrUnhostedBehavior) {
			t.Errorf("%s: RegisterBehavior = %v, want ErrUnhostedBehavior", name, err)
		}
	}
}

// A behavior joins the phases' queries when they are built, so one arriving
// afterwards would silently never run — it is refused instead.
func TestRegisterBehavior_RefusesOneThatComesTooLate(t *testing.T) {
	ecs := goke.New()
	engine := collisions.New(testSpace(t), ecs, testProbeMargin)
	ecs.Setup()
	engine.RegSystems(ecs)

	if err := engine.RegisterBehavior(bulletsAgainstTargets(func(collisions.Meeting) {})[0]); !errors.Is(err, plugin.ErrHostBuilt) {
		t.Errorf("RegisterBehavior after the systems were built = %v, want ErrHostBuilt", err)
	}
}

// Several behaviors go in at once, and the first the plugin cannot host stops
// the lot: what came before it is in, what came after never runs.
func TestRegisterBehavior_StopsAtTheFirstItCannotHost(t *testing.T) {
	var before, after int
	var refused error

	meetWith(t, func(engine registrar) {
		refused = engine.RegisterBehavior(
			plugin.Between[bullet, target](func(plugin.Tick, collisions.Meeting) { before++ }),
			"not a behavior at all",
			plugin.Between[bullet, target](func(plugin.Tick, collisions.Meeting) { after++ }),
		)
	}, &tagged{x: 100, bullet: true}, &tagged{x: 105, target: true})

	if !errors.Is(refused, plugin.ErrUnhostedBehavior) {
		t.Errorf("RegisterBehavior = %v, want ErrUnhostedBehavior", refused)
	}
	if before != 1 || after != 0 {
		t.Errorf("behaviors ran (before %d, after %d), want (1, 0)", before, after)
	}
}
