package gokebiten

import (
	"errors"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/render"
)

type testResourceA struct{ N int }
type testResourceB struct{ S string }

type stubPlugin struct {
	name      string
	installed int
	installFn func(ctx *GameCtx) error
}

func (p *stubPlugin) Name() string { return p.name }
func (p *stubPlugin) Install(ctx *GameCtx) error {
	p.installed++
	if p.installFn != nil {
		return p.installFn(ctx)
	}
	return nil
}
func (p *stubPlugin) RunPlan(goke.RunCtx, time.Duration) {}
func (p *stubPlugin) Renderer() render.Renderer          { return nil }
func (p *stubPlugin) EventHandler() control.EventHandler { return nil }

func TestGame_UsePlugin_InstallsOnce(t *testing.T) {
	game := NewGame(&GameProps{})
	p := &stubPlugin{name: "test.plugin"}

	if err := game.UsePlugin(p); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}
	if p.installed != 1 {
		t.Errorf("installed = %d, want 1", p.installed)
	}
}

func TestGame_UsePlugin_DuplicateNameRejected(t *testing.T) {
	game := NewGame(&GameProps{})
	if err := game.UsePlugin(&stubPlugin{name: "test.plugin"}); err != nil {
		t.Fatalf("first UsePlugin: %v", err)
	}

	err := game.UsePlugin(&stubPlugin{name: "test.plugin"})
	if err == nil {
		t.Fatal("expected an error installing a second plugin with the same Name")
	}
}

func TestGame_Init_RejectsSecondCall(t *testing.T) {
	game := NewGame(&GameProps{})
	noop := func(ctx *GameCtx) error { return nil }

	if err := game.Init(noop); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if err := game.Init(noop); err == nil {
		t.Fatal("expected a second Init call to be rejected")
	}
}

func TestGame_UsePlugin_InstallErrorPropagates(t *testing.T) {
	game := NewGame(&GameProps{})
	p := &stubPlugin{
		name: "test.failing",
		installFn: func(ctx *GameCtx) error {
			_, ok := ctx.Resources.TryGet[*testResourceA]()
			if !ok {
				return errors.New("missing testResourceA")
			}
			return nil
		},
	}

	if err := game.UsePlugin(p); err == nil {
		t.Fatal("expected Install's error to propagate from UsePlugin")
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

func TestGameCtx_Setup_EvaluatesSetupSystemsLazily(t *testing.T) {
	game := NewGame(&GameProps{})
	ctx := plugins.NewGameCtx(
		game.resources, game.ecs, game.step,
		func(v any) { game.pluginManager.track(v) },
		func(producer func() []goke.System) { game.pendingSetup = append(game.pendingSetup, producer) },
	)
	stub := &stubSetupProvider{}
	ctx.Setup(stub)

	if stub.callCount != 0 {
		t.Fatalf("SetupSystems called %d times by Setup(), want 0 (must stay lazy)", stub.callCount)
	}

	game.flushPendingSetup()
	if stub.callCount != 1 {
		t.Errorf("SetupSystems called %d times after flushPendingSetup(), want 1", stub.callCount)
	}
}

func TestGame_UsePlugin_ResolvesOutOfOrderViaNotReady(t *testing.T) {
	game := NewGame(&GameProps{})
	consumer := &stubPlugin{
		name: "test.consumer",
		installFn: func(ctx *GameCtx) error {
			_, err := ctx.Require[*testResourceA]()
			return err
		},
	}
	producer := &stubPlugin{
		name: "test.producer",
		installFn: func(ctx *GameCtx) error {
			ctx.Provide(&testResourceA{N: 1})
			return nil
		},
	}

	if err := game.UsePlugin(consumer); err != nil {
		t.Fatalf("UsePlugin(consumer): %v", err)
	}
	if consumer.installed != 1 {
		t.Fatalf("consumer.installed = %d, want 1 (attempted once, even though not ready)", consumer.installed)
	}

	if err := game.UsePlugin(producer); err != nil {
		t.Fatalf("UsePlugin(producer): %v", err)
	}
	if consumer.installed != 3 {
		t.Errorf("consumer.installed = %d, want 3 (retried within this call: once before producer, once after)", consumer.installed)
	}
}

func TestGame_Run_FailsWhenPluginNeverBecomesReady(t *testing.T) {
	game := NewGame(&GameProps{})
	consumer := &stubPlugin{
		name: "test.consumer",
		installFn: func(ctx *GameCtx) error {
			_, err := ctx.Require[*testResourceA]()
			return err
		},
	}
	if err := game.UsePlugin(consumer); err != nil {
		t.Fatalf("UsePlugin: %v", err)
	}

	err := game.pluginManager.finalizePending()
	if err == nil {
		t.Fatal("expected finalizePending to fail — nobody ever provides *testResourceA")
	}
}

func TestGame_UsePlugin_PanicsWhenPluginWritesBeforeNotReady(t *testing.T) {
	game := NewGame(&GameProps{})
	bad := &stubPlugin{
		name: "test.bad",
		installFn: func(ctx *GameCtx) error {
			ctx.Provide(&testResourceB{S: "too early"})
			_, err := ctx.Require[*testResourceA]()
			return err
		},
	}

	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic — plugin wrote before confirming its dependency")
		}
	}()
	_ = game.UsePlugin(bad)
}

func TestGameCtx_InsertResource_VisibleToLaterPlugins(t *testing.T) {
	game := NewGame(&GameProps{})
	first := &stubPlugin{
		name: "test.provider",
		installFn: func(ctx *GameCtx) error {
			ctx.Provide(&testResourceA{N: 9})
			return nil
		},
	}
	var gotN int
	second := &stubPlugin{
		name: "test.consumer",
		installFn: func(ctx *GameCtx) error {
			gotN = ctx.Resources.Get[*testResourceA]().N
			return nil
		},
	}

	if err := game.UsePlugin(first); err != nil {
		t.Fatalf("UsePlugin(first): %v", err)
	}
	if err := game.UsePlugin(second); err != nil {
		t.Fatalf("UsePlugin(second): %v", err)
	}
	if gotN != 9 {
		t.Errorf("second plugin read N = %d, want 9", gotN)
	}
}
