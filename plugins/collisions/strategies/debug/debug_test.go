package debug_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/debug"
	"github.com/kjkrol/uid"
)

// run ticks the behavior over two entities that each recorded the other.
func run(t *testing.T, opts ...debug.Option) (idA, idB uid.UID64) {
	t.Helper()
	var contactsComp goke.Comp[collisions.Contacts]

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&contactsComp)
		f.Create(2)
		f.Next()
		idA, idB = f.IDs[0], f.IDs[1]
		slice := contactsComp.Slice(&f.Cursor)
		slice[0].Items[0] = collisions.Contact{Other: idB, Impact: 2.5}
		slice[0].Count = 1
		slice[1].Items[0] = collisions.Contact{Other: idA, Impact: 2.5}
		slice[1].Count = 1
	}})

	handle := ecs.RegSys(debug.New(opts...))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)
	return idA, idB
}

func TestBehavior_LogsEachPairOnce(t *testing.T) {
	var out bytes.Buffer

	idA, idB := run(t, debug.WithWriter(&out))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("logged %d lines, want 1 for a contact both sides recorded: %q", len(lines), out.String())
	}
	for _, want := range []string{fmt.Sprint(idA), fmt.Sprint(idB), "2.50"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("line %q is missing %q", lines[0], want)
		}
	}
}

func TestBehavior_WithFormat_ReplacesTheLine(t *testing.T) {
	var out bytes.Buffer

	run(t, debug.WithWriter(&out), debug.WithFormat(func(_ uid.UID64, c collisions.Contact) string {
		return "struck with " + string(rune('0'+int(c.Impact)))
	}))

	if got := strings.TrimSpace(out.String()); got != "struck with 2" {
		t.Errorf("line = %q, want the custom format", got)
	}
}
