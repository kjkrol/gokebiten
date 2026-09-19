package stats_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/stats"
	"github.com/kjkrol/uid"
)

// run ticks the behavior over two entities that each recorded the other, plus
// however many extra contacts the caller adds to the first one.
func run(t *testing.T, s *stats.Stats, extra ...uid.UID64) {
	t.Helper()
	var contactsComp goke.Comp[collisions.Contacts]

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&contactsComp)
		f.Create(2)
		f.Next()
		idA, idB := f.IDs[0], f.IDs[1]
		slice := contactsComp.Slice(&f.Cursor)
		slice[0].Items[0] = collisions.Contact{Other: idB}
		slice[0].Count = 1
		slice[1].Items[0] = collisions.Contact{Other: idA}
		slice[1].Count = 1
		for _, id := range extra {
			slice[0].Items[slice[0].Count] = collisions.Contact{Other: id}
			slice[0].Count++
		}
	}})

	handle := ecs.RegSys(stats.New(s))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)
}

// Both sides of a contact record it, so counting every record would double
// every collision.
func TestBehavior_CountsEachPairOnce(t *testing.T) {
	var s stats.Stats

	run(t, &s)

	if s.Counter != 1 {
		t.Errorf("Counter = %d, want 1 for a contact both sides recorded", s.Counter)
	}
}

func TestBehavior_AccumulatesAcrossTicks(t *testing.T) {
	s := stats.Stats{Counter: 5}

	run(t, &s)

	if s.Counter != 6 {
		t.Errorf("Counter = %d, want 6 — the count carries over until Reset", s.Counter)
	}
}
