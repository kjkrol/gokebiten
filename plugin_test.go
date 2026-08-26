package gokebiten

import (
	"errors"
	"testing"

	"github.com/kjkrol/goke/v3"
)

type stubPlugin struct {
	name      string
	installed int
	installFn func(ctx *PluginContext) error
}

func (p *stubPlugin) Name() string { return p.name }
func (p *stubPlugin) Install(ctx *PluginContext) error {
	p.installed++
	if p.installFn != nil {
		return p.installFn(ctx)
	}
	return nil
}

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

func TestGame_UsePlugin_InstallErrorPropagates(t *testing.T) {
	game := NewGame(&GameProps{})
	p := &stubPlugin{
		name: "test.failing",
		installFn: func(ctx *PluginContext) error {
			_, ok := ctx.Resources.TryGetResource[*testResourceA]()
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

func TestGame_Setup_EvaluatesSetupSystemsLazily(t *testing.T) {
	game := NewGame(&GameProps{})
	stub := &stubSetupProvider{}
	game.setup(stub)

	if stub.callCount != 0 {
		t.Fatalf("SetupSystems called %d times by setup(), want 0 (must stay lazy)", stub.callCount)
	}

	game.flushPendingSetup()
	if stub.callCount != 1 {
		t.Errorf("SetupSystems called %d times after flushPendingSetup(), want 1", stub.callCount)
	}
}

func TestPluginContext_InsertResource_VisibleToLaterPlugins(t *testing.T) {
	game := NewGame(&GameProps{})
	first := &stubPlugin{
		name: "test.provider",
		installFn: func(ctx *PluginContext) error {
			ctx.Resources.InsertResource(&testResourceA{N: 9})
			return nil
		},
	}
	var gotN int
	second := &stubPlugin{
		name: "test.consumer",
		installFn: func(ctx *PluginContext) error {
			gotN = ctx.Resources.GetResource[*testResourceA]().N
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
