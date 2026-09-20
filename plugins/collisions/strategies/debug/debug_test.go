package debug_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/debug"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	gokgspatial "github.com/kjkrol/gokg/spatial"
	"github.com/kjkrol/uid"
)

// collide runs the real collision engine for one tick over two elastic boxes
// closing head-on, with the debug behavior hosted in it.
func collide(t *testing.T, opts ...debug.Option) (idA, idB uid.UID64) {
	t.Helper()
	space, err := gokg.NewSpace(gokg.Config{
		Width: 1000, Height: 1000,
		BucketSize: gokgspatial.ResolutionFrom(64), BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("gokg.NewSpace: %v", err)
	}

	ecs := goke.New()
	engine := collisions.New(space, ecs, 10)
	if err := engine.RegisterBehavior(plugin.Between[plugin.Anything, plugin.Anything](debug.Log(opts...))); err != nil {
		t.Fatalf("RegisterBehavior: %v", err)
	}

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var base goke.Comp[world.Base]
		var coll goke.Comp[collisions.Collision]
		var physics goke.Comp[collisions.Physics]
		f := si.NewFactory(&base, &coll, &physics)
		f.Create(2)
		f.Next()
		idA, idB = f.IDs[0], f.IDs[1]
		for i, x := range []float64{100, 105} {
			box := plane.NewAABB(geom.NewVec(x, 100), 10, 10)
			base.Slice(&f.Cursor)[i].Pos = world.Position{AABB: box}
			physics.Slice(&f.Cursor)[i] = collisions.Physics{Restitution: 1}
			space.Insert(f.IDs[i], box)
			space.SetCapabilities(f.IDs[i], collisions.CanCollide)
		}
		base.Slice(&f.Cursor)[0].Vel.SetDelta(geom.NewVec(5, 0))
		base.Slice(&f.Cursor)[1].Vel.SetDelta(geom.NewVec(-5, 0))
		space.Flush(nil)
	}})
	engine.RegSystems(ecs)
	ecs.SetPlan(engine.RunPlan)
	ecs.Tick(time.Millisecond)
	return idA, idB
}

func TestBehavior_LogsEachContactOnce(t *testing.T) {
	var out bytes.Buffer

	idA, idB := collide(t, debug.WithWriter(&out))

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("logged %d lines, want 1 for one contact: %q", len(lines), out.String())
	}
	for _, want := range []string{fmt.Sprint(idA), fmt.Sprint(idB), "10.00"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("line %q is missing %q", lines[0], want)
		}
	}
}

func TestBehavior_WithFormat_ReplacesTheLine(t *testing.T) {
	var out bytes.Buffer

	collide(t, debug.WithWriter(&out), debug.WithFormat(func(m collisions.Meeting) string {
		return fmt.Sprintf("struck with %.0f", m.Impact)
	}))

	if got := strings.TrimSpace(out.String()); got != "struck with 10" {
		t.Errorf("line = %q, want the custom format", got)
	}
}
