package main

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/vision/strategies/hunt"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// stageInit is a game.Initializer that drives the real Stage without a window:
// queue every Use/UseModule/Setup call, then flush them through one ecs.Setup,
// exactly as the engine does when it enters a Stage. Scene.Layers() is left
// out — it builds an Atlas, which needs a graphics context.
type stageInit struct {
	ecs     *goke.ECS
	world   *world.Plugin
	tracked []any
	pending []func() []goke.System
	tps     game.TPS
}

var _ game.Initializer = (*stageInit)(nil)

func (c *stageInit) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(*goke.SysInit) { m.RegSystems(c.ecs) }}
	c.tracked = append(c.tracked, m)
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}

func (c *stageInit) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.tracked = append(c.tracked, p)
		c.pending = append(c.pending, p.SetupSystems)
	}
}

func (c *stageInit) RegSys(factory func() goke.System) goke.Runnable { return c.ecs.RegSys(factory()) }
func (c *stageInit) ECS() *goke.ECS                                  { return c.ecs }
func (c *stageInit) TPS() *game.TPS                                  { return &c.tps }

func (c *stageInit) Use(p plugin.Plugin) error {
	c.tracked = append(c.tracked, p)
	return p.Install(c)
}

func (c *stageInit) Track(s plugin.Serializable) error {
	c.tracked = append(c.tracked, s)
	return nil
}

func (c *stageInit) UseWorld(cfg world.Config) *world.Plugin {
	cfg.Camera.ViewportWidth = ScreenWidth
	cfg.Camera.ViewportHeight = ScreenHeight
	c.world = world.NewPlugin(cfg)
	c.tracked = append(c.tracked, c.world)
	if err := c.world.Install(c); err != nil {
		panic(err)
	}
	return c.world
}

// buildStage runs the fresh-spawn half of entering a Stage: Init, Spawn,
// Populate, SetPlan, one flushing ecs.Setup.
func buildStage(t *testing.T) (*goke.ECS, *mainStage) {
	t.Helper()

	stage := &mainStage{avoiding: true}
	ctx := &stageInit{ecs: goke.New()}
	if err := stage.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := stage.Spawn(); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	for _, v := range ctx.tracked {
		if p, ok := v.(plugin.Populator); ok {
			if err := p.Populate(); err != nil {
				t.Fatalf("Populate: %v", err)
			}
		}
	}
	ctx.ecs.SetPlan(stage.Update)

	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)

	return ctx.ecs, stage
}

// A catch has to cost the prey its life, and that takes the whole chain: the
// hunter carrying Contacts and a sensor body, the eat behaviour registered, and
// world.Despawn clearing both the ECS and the index. Leave out any one of them
// and the demo still runs, still builds, and the hunter merely drifts through
// everyone forever — so this stages a catch rather than waiting for one.
func TestStage_HunterEatsWhatItCatches(t *testing.T) {
	ecs, stage := buildStage(t)
	view := bodies(ecs)

	if len(ids(view.prey)) != PreyCount {
		t.Fatalf("stage spawned %d prey, want %d", len(ids(view.prey)), PreyCount)
	}
	caught := placeOnPrey(t, stage, view)

	for range 3 {
		ecs.Tick(time.Second / TPS)
	}

	if left := ids(view.prey); left[caught] {
		t.Errorf("prey %v survived being caught", caught)
	} else if len(left) != PreyCount-1 {
		t.Errorf("%d prey left, want %d — exactly the caught one gone", len(left), PreyCount-1)
	}
	if got, want := stage.world.Res.Telemetry.Count, PreyCount; got != want {
		t.Errorf("Telemetry.Count = %d, want %d (the prey and the hunter, one prey short)", got, want)
	}
}

// placeOnPrey drops the hunter straight onto the first prey — in the ECS and in
// the spatial index the broad phase probes — and returns that prey's id.
func placeOnPrey(t *testing.T, stage *mainStage, view bodyView) uid.UID64 {
	t.Helper()
	var target world.Position
	var caught uid.UID64
	found := false
	view.prey.All()
	for view.prey.Next() && !found {
		cursor := view.prey.Cursor()
		target, caught, found = view.preyPos.Slice(cursor)[0], cursor.IDs[0], true
	}
	if !found {
		t.Fatal("no prey to place the hunter on")
	}

	view.hunters.All()
	for view.hunters.Next() {
		cursor := view.hunters.Cursor()
		view.hunterPos.Slice(cursor)[0] = target
		stage.world.Space().Reindex(cursor.IDs[0], target.AABB)
	}
	stage.world.Space().Flush(nil)
	return caught
}

// bodyView is a live view of the hunters, the prey, and where each of them is.
type bodyView struct {
	hunters, prey      *goke.Query
	hunterPos, preyPos *goke.Comp[world.Position]
}

func bodies(ecs *goke.ECS) bodyView {
	view := bodyView{hunterPos: new(goke.Comp[world.Position]), preyPos: new(goke.Comp[world.Position])}
	ecs.RegSys(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		view.hunters = si.NewQueryBuilder(view.hunterPos).Include(goke.Include[hunt.Predator]()).Build()
		view.prey = si.NewQueryBuilder(view.preyPos).Include(goke.Include[hunt.Prey]()).Build()
	}})
	return view
}

func ids(q *goke.Query) map[uid.UID64]bool {
	found := map[uid.UID64]bool{}
	q.All()
	for q.Next() {
		for _, id := range q.Cursor().IDs {
			found[id] = true
		}
	}
	return found
}
