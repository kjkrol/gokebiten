package main

import (
	"testing"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/vision/behavior"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

// stageInit is a game.Initializer that drives the real Stage without a window;
// Scene.Layers() is left out.
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

// buildStage runs the fresh-spawn half of entering a Stage: Init, Spawn, Populate, Setup.
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

// placeOnPrey drops the hunter onto the first prey, in the ECS and the index, and returns its id.
func placeOnPrey(t *testing.T, stage *mainStage, view bodyView) uid.UID64 {
	t.Helper()
	var target world.Position
	var caught uid.UID64
	found := false
	view.prey.All()
	for view.prey.Next() && !found {
		cursor := view.prey.Cursor()
		target, caught, found = view.preyBase.Slice(cursor)[0].Pos, cursor.IDs[0], true
	}
	if !found {
		t.Fatal("no prey to place the hunter on")
	}

	view.hunters.All()
	for view.hunters.Next() {
		cursor := view.hunters.Cursor()
		stage.world.Space().MoveTo(cursor.IDs[0], &view.hunterBase.Slice(cursor)[0].Pos.AABB, target.TopLeft)
	}
	stage.world.Space().Flush(nil)
	return caught
}

// bodyView is a live view of the hunters, the prey, and where each of them is.
type bodyView struct {
	hunters, prey        *goke.Query
	hunterBase, preyBase *goke.Comp[world.Base]
}

func bodies(ecs *goke.ECS) bodyView {
	view := bodyView{
		hunterBase: new(goke.Comp[world.Base]),
		preyBase:   new(goke.Comp[world.Base]),
	}
	ecs.RegSys(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		view.hunters = si.NewQueryBuilder(view.hunterBase).Include(goke.Include[behavior.Predator]()).Build()
		view.prey = si.NewQueryBuilder(view.preyBase).Include(goke.Include[behavior.Prey]()).Build()
	}})
	return view
}

func TestStage_PreyTurnsAwayFromTheHunterItSees(t *testing.T) {
	ecs, stage := buildStage(t)
	view := bodies(ecs)

	watched, course := placeHunterAhead(t, stage, view, 60)

	const ticks = 14
	for range ticks {
		ecs.Tick(time.Second / TPS)
	}

	now, alive := headingOf(view, watched)
	if !alive {
		t.Fatalf("the watched prey was gone within %d ticks — it never got the chance to run", ticks)
	}
	if along := now.X*course.X + now.Y*course.Y; along > 0.5 {
		t.Errorf("heading %v after %d ticks of looking at the hunter, started %v — want it well into turning away", now, ticks, course)
	}
}

// placeHunterAhead puts the hunter dead ahead of the first prey; returns its id and course.
func placeHunterAhead(t *testing.T, stage *mainStage, view bodyView, distance float64) (uid.UID64, geom.Vec) {
	t.Helper()
	var watched uid.UID64
	var from world.Position
	var course geom.Vec
	found := false
	view.prey.All()
	for view.prey.Next() && !found {
		cursor := view.prey.Cursor()
		seen := view.preyBase.Slice(cursor)[0]
		watched, from, course, found = cursor.IDs[0], seen.Pos, seen.Vel.Dir, true
	}
	if !found {
		t.Fatal("no prey to put the hunter in front of")
	}

	ahead := geom.NewVec(
		float64(from.TopLeft.X)+course.X*distance,
		float64(from.TopLeft.Y)+course.Y*distance,
	)

	view.hunters.All()
	for view.hunters.Next() {
		cursor := view.hunters.Cursor()
		stage.world.Space().MoveTo(cursor.IDs[0], &view.hunterBase.Slice(cursor)[0].Pos.AABB, ahead)
	}
	stage.world.Space().Flush(nil)
	return watched, course
}

func headingOf(view bodyView, id uid.UID64) (geom.Vec, bool) {
	view.prey.All()
	for view.prey.Next() {
		cursor := view.prey.Cursor()
		for i, got := range cursor.IDs {
			if got == id {
				return view.preyBase.Slice(cursor)[i].Vel.Dir, true
			}
		}
	}
	return geom.Vec{}, false
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
