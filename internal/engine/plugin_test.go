package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

func testProps() *Props {
	return &Props{World: world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}}
}

type stubPlugin struct {
	name         string
	installed    int
	installFn    func(ctx plugin.Installer) error
	serializable plugin.Serializable
}

func (p *stubPlugin) Name() string { return p.name }
func (p *stubPlugin) Install(ctx plugin.Installer) error {
	p.installed++
	if p.installFn != nil {
		return p.installFn(ctx)
	}
	return nil
}
func (p *stubPlugin) RunPlan(goke.RunCtx, time.Duration) {}
func (p *stubPlugin) WithRenderer(render.AtlasSource)    {}
func (p *stubPlugin) Renderer() render.Renderer          { return nil }
func (p *stubPlugin) EventHandler() control.EventHandler { return nil }
func (p *stubPlugin) Serializable() plugin.Serializable  { return p.serializable }

// stubGame is a minimal Game for testing Engine/Initializer.
type stubGame struct {
	initFn func(ctx game.Initializer) error
}

func (g *stubGame) Init(ctx game.Initializer) error {
	if g.initFn != nil {
		return g.initFn(ctx)
	}
	return nil
}
func (g *stubGame) Restore(game.Persistence) (bool, error)          { return false, nil }
func (g *stubGame) Spawn() ([]world.Batch, error)                   { return nil, nil }
func (g *stubGame) Update(goke.RunCtx, time.Duration)               {}
func (g *stubGame) Draw(game.Runtime) []func() render.Renderer      { return nil }
func (g *stubGame) HandleEvents(*control.InputEvents, game.Runtime) {}

func newTestEngine(initFn func(ctx game.Initializer) error) *Engine {
	return NewEngine(testProps(), &stubGame{initFn: initFn})
}

func TestInitializer_Use_InstallsOnce(t *testing.T) {
	p := &stubPlugin{name: "test.plugin"}
	eng := newTestEngine(func(ctx game.Initializer) error { return ctx.Use(p) })

	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.installed != 1 {
		t.Errorf("installed = %d, want 1", p.installed)
	}
}

func TestInitializer_Use_DuplicateNameRejected(t *testing.T) {
	eng := newTestEngine(func(ctx game.Initializer) error {
		if err := ctx.Use(&stubPlugin{name: "test.plugin"}); err != nil {
			return err
		}
		return ctx.Use(&stubPlugin{name: "test.plugin"})
	})

	if err := eng.Init(); err == nil {
		t.Fatal("expected an error using a second plugin with the same Name")
	}
}

func TestEngine_Init_PropagatesGameInitError(t *testing.T) {
	wantErr := errors.New("test: Game.Init failed")
	eng := newTestEngine(func(ctx game.Initializer) error { return wantErr })

	if err := eng.Init(); err != wantErr {
		t.Fatalf("Init() = %v, want %v", err, wantErr)
	}
}

type stubSetupProvider struct {
	callCount int
	systems   []goke.System
}

func (s *stubSetupProvider) SetupSystems() []goke.System {
	s.callCount++
	return s.systems
}

func TestEngine_Init_EvaluatesSetupSystemsLazily(t *testing.T) {
	stub := &stubSetupProvider{}
	var duringInit int
	eng := newTestEngine(func(ctx game.Initializer) error {
		ctx.Setup(stub)
		duringInit = stub.callCount
		return nil
	})

	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if duringInit != 0 {
		t.Fatalf("SetupSystems called %d times during Game.Init, want 0 (must stay lazy)", duringInit)
	}
	if stub.callCount != 1 {
		t.Errorf("SetupSystems called %d times after engine.Init(), want 1", stub.callCount)
	}
}

func TestInitializer_Use_RegistersSerializableByName(t *testing.T) {
	p := &stubPlugin{name: "test.resource", serializable: &testPersisted{N: 7}}
	eng := newTestEngine(func(ctx game.Initializer) error { return ctx.Use(p) })

	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	targets, ok := eng.resources.persisted()["test.resource"]
	if !ok || len(targets) != 1 || *(targets[0].(*int)) != 7 {
		t.Errorf("expected stubPlugin's Serializable() registered under its Name(), got %+v, ok=%v", targets, ok)
	}
}

type testPersisted struct{ N int }

func (p *testPersisted) Persisted() []any { return []any{&p.N} }

// stubBuiltinPlugin implements Builtin, so Use must reject it.
type stubBuiltinPlugin struct{ stubPlugin }

func (*stubBuiltinPlugin) Builtin() {}

func TestInitializer_Use_RejectsBuiltinPlugin(t *testing.T) {
	p := &stubBuiltinPlugin{stubPlugin: stubPlugin{name: "test.builtin"}}
	eng := newTestEngine(func(ctx game.Initializer) error { return ctx.Use(p) })

	if err := eng.Init(); err == nil {
		t.Fatal("expected Use to reject a Plugin implementing Builtin")
	}
	if p.installed != 0 {
		t.Errorf("installed = %d, want 0 (rejected before Install)", p.installed)
	}
}

// TestEngine_Init_WorldViewportDefaultsToScreenSize guards that Engine
// fills in the built-in world's camera viewport from Props.ScreenWidth/
// ScreenHeight when the game leaves World.Camera.Viewport* unset — otherwise the
// camera's pannable window defaults to the world's own size, leaving
// zero room to pan or zoom out regardless of screen size.
func TestEngine_Init_WorldViewportDefaultsToScreenSize(t *testing.T) {
	props := &Props{ScreenWidth: 200, ScreenHeight: 150, World: world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}}
	var got camera.Camera
	eng := NewEngine(props, &stubGame{initFn: func(ctx game.Initializer) error {
		got = ctx.World().Camera()
		return nil
	}})
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	b := got.Bounds()
	if w, h := b.BottomRight.X-b.TopLeft.X, b.BottomRight.Y-b.TopLeft.Y; w != 200 || h != 150 {
		t.Errorf("Camera().Bounds() size = %dx%d, want 200x150 (screen size, not world size)", w, h)
	}
}

func TestInitializer_World_ReturnsInstalledInstance(t *testing.T) {
	var got *world.Plugin
	eng := newTestEngine(func(ctx game.Initializer) error {
		got = ctx.World()
		return nil
	})

	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got == nil {
		t.Fatal("expected ctx.World() to return the engine's built-in world.Plugin, got nil")
	}
}
