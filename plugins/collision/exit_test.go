package collision_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

type pushedOut struct {
	x    float64
	wall bool
}

func TestCollision_ABoxPushedThroughAnOpenEdgeIsReportedToTheWorld(t *testing.T) {
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000, Edges: aabbworld.OpenX},
		Entities: world.EntitiesCfg{MaxCount: 4, MinSize: 10, MaxSize: 10},
	})
	var left []uid.UID64
	w.OnExit(func(_ plugin.Tick, id uid.UID64) { left = append(left, id) })

	c := collision.NewPlugin(w)
	ctx := &installCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		t.Fatalf("world Install: %v", err)
	}
	if err := c.Install(ctx); err != nil {
		t.Fatalf("collision Install: %v", err)
	}

	boxes := kind.Define[pushedOut](w.Kinds(), "box", kind.Spec{
		kind.Load(func(b pushedOut) world.Position { return posAt(b.x, 500, 10, 10) }),
		kind.Const(world.Velocity{}),
		kind.Const(collision.Collider{}),
		kind.Load(func(b pushedOut) collision.Physics {
			if b.wall {
				return collision.Physics{Mass: math.Inf(1)}
			}
			return collision.Physics{}
		}),
	})
	w.Seed(boxes.Entry(pushedOut{x: 0, wall: true}), boxes.Entry(pushedOut{x: -9}))
	if err := w.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		w.RunPlan(rc, d)
		c.RunPlan(rc, d)
		rc.Sync()
	})
	for range 3 {
		ctx.ecs.Tick(time.Second / 60)
	}

	if len(left) != 1 {
		t.Fatalf("OnExit heard %v, want exactly the box the wall pushed out", left)
	}
}
