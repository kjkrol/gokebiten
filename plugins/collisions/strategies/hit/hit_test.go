package hit_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/hit"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

const fallback = time.Second

// entity is the pair of components the behavior works on, as one test fixture.
type entity struct {
	mark     hit.Mark
	contacts collisions.Contacts
}

func run(t *testing.T, b *hit.Behavior, entities ...entity) []hit.Mark {
	t.Helper()
	var markComp goke.Comp[hit.Mark]
	var contactsComp goke.Comp[collisions.Contacts]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		for _, e := range entities {
			f := si.NewFactory(&markComp, &contactsComp)
			f.Create(1)
			f.Next()
			markComp.Slice(&f.Cursor)[0] = e.mark
			contactsComp.Slice(&f.Cursor)[0] = e.contacts
		}
		q = si.NewQueryBuilder(&markComp).Build()
	}})

	handle := ecs.RegSys(b)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	var got []hit.Mark
	q.All()
	for q.Next() {
		got = append(got, markComp.Slice(q.Cursor())...)
	}
	return got
}

func struck() collisions.Contacts {
	var c collisions.Contacts
	c.Items[0] = collisions.Contact{Other: uid.UID64(1), Impact: 3}
	c.Count = 1
	return c
}

// An entity's own Duration wins; the behavior's is what an entity that sets
// none falls back to.
func TestBehavior_MarksForTheEntitysOwnDuration(t *testing.T) {
	const own = 250 * time.Millisecond
	before := time.Now()

	got := run(t, hit.New(fallback),
		entity{mark: hit.Mark{Duration: own}, contacts: struck()},
		entity{contacts: struck()},
	)

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

func TestBehavior_NoContact_LeavesAFreshMarkAlone(t *testing.T) {
	fresh := time.Now().Add(time.Hour).UnixNano()

	got := run(t, hit.New(fallback), entity{mark: hit.Mark{ExpiresAtNano: fresh}})

	if len(got) != 1 || got[0].ExpiresAtNano != fresh {
		t.Errorf("mark = %+v, want the untouched stamp %v", got[0], fresh)
	}
}

// Clearing the lapsed stamp here is what lets a renderer read Active instead
// of asking the clock per entity.
func TestBehavior_NoContact_ClearsALapsedMark(t *testing.T) {
	got := run(t, hit.New(fallback), entity{mark: hit.Mark{ExpiresAtNano: time.Now().Add(-time.Hour).UnixNano()}})

	if len(got) != 1 || got[0].Active() {
		t.Errorf("mark = %+v, want it cleared once its time had passed", got[0])
	}
}

func TestOverlay_DrawsOnlyWhileTheMarkIsActive(t *testing.T) {
	base := world.Appearance{SpriteID: 1}
	flash := world.Appearance{SpriteID: 2}
	overlay := hit.Overlay(flash)

	active := overlay.Resolve([]world.Appearance{base}, hit.Mark{ExpiresAtNano: 1})
	if len(active) != 2 || active[1] != flash {
		t.Errorf("layers while active = %v, want the flash on top of %v", active, base)
	}
	idle := overlay.Resolve([]world.Appearance{base}, hit.Mark{})
	if len(idle) != 1 || idle[0] != base {
		t.Errorf("layers while idle = %v, want just %v", idle, base)
	}
}
