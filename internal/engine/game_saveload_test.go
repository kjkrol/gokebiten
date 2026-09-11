package engine_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

func newTestWorldPlugin() *world.Plugin {
	return world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000},
		Entities: world.EntitiesCfg{MaxCount: 10, MinSize: 1, MaxSize: 100},
	})
}

type saveTestState struct{ N int }
type saveTestResourceB struct{ S string }

// ecsAccessor captures ctx.ECS() during Install and queues an optional
// setup callback into the same deferred ecs.Setup batch — the only way to
// reach *goke.ECS/run ad-hoc SysInit logic now that it's gated behind Plugin.
type ecsAccessor struct {
	ecs   *goke.ECS
	setup func(*goke.SysInit)
}

func (a *ecsAccessor) Name() string { return "test.ecs-accessor" }
func (a *ecsAccessor) Install(ctx plugin.Installer) error {
	a.ecs = ctx.ECS()
	if a.setup != nil {
		ctx.Setup(a)
	}
	return nil
}
func (a *ecsAccessor) SetupSystems() []goke.System {
	return []goke.System{goke.SystemFn{OnInit: a.setup}}
}
func (a *ecsAccessor) RunPlan(goke.RunCtx, time.Duration)             {}
func (a *ecsAccessor) WithRenderer(camera.Camera, render.AtlasSource) {}
func (a *ecsAccessor) Renderer() render.Renderer                      { return nil }
func (a *ecsAccessor) EventHandler() control.EventHandler             { return nil }
func (a *ecsAccessor) Serializable() plugin.Serializable              { return nil }

// saveLoadTestGame wires newTestWorldPlugin + ecsAccessor for the round-trip test below.
type saveLoadTestGame struct {
	acc      *ecsAccessor
	setup    func(*goke.SysInit)
	loadFrom string
	loadArgs []any
}

func (g *saveLoadTestGame) Init(ctx game.Initializer) error {
	if err := ctx.Use(newTestWorldPlugin()); err != nil {
		return err
	}
	g.acc = &ecsAccessor{setup: g.setup}
	if err := ctx.Use(g.acc); err != nil {
		return err
	}
	if g.loadFrom != "" {
		return ctx.Runtime().Persistence().Load(g.loadFrom, "", g.loadArgs...)
	}
	return nil
}
func (g *saveLoadTestGame) RunPlan(goke.RunCtx, time.Duration)              {}
func (g *saveLoadTestGame) Layers(game.Runtime) []func() render.Renderer    { return nil }
func (g *saveLoadTestGame) HandleEvents(*control.InputEvents, game.Runtime) {}

// TestGame_SaveLoad_RoundTrip guards that Engine.Persistence.Save/Load correctly delegate to the engine's own ECS and resources.
func TestGame_SaveLoad_RoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"

	var appearance goke.Comp[world.Appearance]
	g := &saveLoadTestGame{setup: func(si *goke.SysInit) {
		f := si.NewFactory(&appearance)
		f.Create(1)
		f.Next()
		appearance.Slice(&f.Cursor)[0] = world.Appearance{SpriteID: 7}
	}}
	eng := engine.NewEngine(&engine.Props{}, g)
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	state := &saveTestState{N: 42}
	extra := &saveTestResourceB{S: "hello"}
	if err := eng.Persistence().Save(basePath, "", state, extra); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var appearance2 goke.Comp[world.Appearance]
	var q *goke.Query
	state2 := &saveTestState{}
	extra2 := &saveTestResourceB{}
	game2 := &saveLoadTestGame{
		setup: func(si *goke.SysInit) {
			q = si.NewQueryBuilder(&appearance2).Build()
		},
		loadFrom: basePath,
		loadArgs: []any{state2, extra2},
	}
	eng2 := engine.NewEngine(&engine.Props{}, game2)
	if err := eng2.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if state2.N != 42 {
		t.Errorf("state2.N after Load = %d, want 42", state2.N)
	}
	if extra2.S != "hello" {
		t.Errorf("extra2.S after Load = %q, want %q", extra2.S, "hello")
	}

	q.All()
	found := false
	for q.Next() {
		for _, a := range appearance2.Slice(q.Cursor()) {
			found = true
			if a.SpriteID != 7 {
				t.Errorf("SpriteID = %d, want 7", a.SpriteID)
			}
		}
	}
	if !found {
		t.Fatal("expected the saved world.Appearance entity to survive the round trip")
	}
}
