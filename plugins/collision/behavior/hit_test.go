package behavior_test

import (
	"testing"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collision"
	"github.com/kjkrol/gokebiten/plugins/collision/behavior"
	"github.com/kjkrol/gokebiten/plugins/world"
)

const fallback = time.Second

// entity is one collidable box that keeps a Mark, at x along a row.
type entity struct {
	x    float64
	mark behavior.HitMark
}

// run hosts the hit behavior in the real collision engine for two ticks and returns the marks.
func run(t *testing.T, entities ...entity) []behavior.HitMark {
	t.Helper()
	space, err := aabbworld.NewSpace(aabbworld.Config{
		Width: 1000, Height: 1000,
		BucketSize: 64, BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("aabbworld.NewSpace: %v", err)
	}

	ecs := goke.New()
	engine := collision.New(space, ecs)
	if err := engine.RegisterBehavior(plugin.Each[behavior.HitMark](behavior.ShowHits(fallback))); err != nil {
		t.Fatalf("RegisterBehavior: %v", err)
	}

	var marks goke.Comp[behavior.HitMark]
	var q *goke.Query
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var base goke.Comp[world.Base]
		var coll goke.Comp[collision.Collider]
		f := si.NewFactory(&base, &coll, &marks)
		f.Create(len(entities))
		f.Next()
		for i, e := range entities {
			box := plane.NewAABB(geom.NewVec(e.x, 100), 10, 10)
			base.Slice(&f.Cursor)[i].Pos = world.Position{AABB: box}
			marks.Slice(&f.Cursor)[i] = e.mark
			space.Insert(f.IDs[i], &box)
		}
		space.Flush(nil)
		q = si.NewQueryBuilder(&marks).Build()
	}})
	engine.RegSystems(ecs)
	ecs.SetPlan(engine.RunPlan)
	ecs.Tick(time.Millisecond)
	ecs.Tick(time.Millisecond)

	var got []behavior.HitMark
	for q.All(); q.Next(); {
		got = append(got, marks.Slice(q.Cursor())...)
	}
	return got
}

func TestShowHits_MarksForTheEntitysOwnDuration(t *testing.T) {
	const own = 250 * time.Millisecond
	before := time.Now()

	got := run(t, entity{x: 100, mark: behavior.HitMark{Duration: own}}, entity{x: 105})

	if len(got) != 2 {
		t.Fatalf("got %d marks, want 2", len(got))
	}
	for i, want := range []time.Duration{own, fallback} {
		if !got[i].Active() {
			t.Fatalf("entity %d was struck but shows no hit", i)
		}
		showing := time.Unix(0, got[i].ExpiresAtNano).Sub(before)
		if showing < want || showing > want+time.Second {
			t.Errorf("entity %d shows its hit for ~%v, want ~%v", i, showing, want)
		}
	}
}

func TestShowHits_NoContact_LeavesAFreshMarkAlone(t *testing.T) {
	fresh := time.Now().Add(time.Hour).UnixNano()

	got := run(t, entity{x: 100, mark: behavior.HitMark{ExpiresAtNano: fresh}})

	if len(got) != 1 || got[0].ExpiresAtNano != fresh {
		t.Errorf("mark = %+v, want the untouched stamp %v", got, fresh)
	}
}

func TestShowHits_NoContact_ClearsALapsedMark(t *testing.T) {
	got := run(t, entity{x: 100, mark: behavior.HitMark{ExpiresAtNano: time.Now().Add(-time.Hour).UnixNano()}})

	if len(got) != 1 || got[0].Active() {
		t.Errorf("mark = %+v, want it cleared once its time had passed", got)
	}
}

func TestHitOverlay_DrawsOnlyWhileTheMarkIsActive(t *testing.T) {
	base := world.Appearance{SpriteID: 1}
	flash := world.Appearance{SpriteID: 2}
	overlay := behavior.HitOverlay(flash)

	active := overlay.Resolve([]world.Appearance{base}, behavior.HitMark{ExpiresAtNano: 1})
	if len(active) != 2 || active[1] != flash {
		t.Errorf("layers while active = %v, want the flash on top of %v", active, base)
	}
	idle := overlay.Resolve([]world.Appearance{base}, behavior.HitMark{})
	if len(idle) != 1 || idle[0] != base {
		t.Errorf("layers while idle = %v, want just %v", idle, base)
	}
}
