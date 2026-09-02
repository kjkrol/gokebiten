package gokebiten

import (
	"testing"

	"github.com/kjkrol/goke/v3"
)

type stubPostLoader struct{ ran bool }

func (s *stubPostLoader) SetupSystems() []goke.System { return nil }
func (s *stubPostLoader) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) { s.ran = true }}
}

func TestPluginManager_PostLoadSystems_RunsTrackedPostLoader(t *testing.T) {
	game := NewGame(&GameProps{})
	stub := &stubPostLoader{}
	game.pluginManager.track(stub)

	systems := game.pluginManager.postLoadSystems()
	if len(systems) != 1 {
		t.Fatalf("postLoadSystems() returned %d systems, want 1", len(systems))
	}

	goke.New().Setup(systems...)
	if !stub.ran {
		t.Error("expected the tracked PostLoader's system to have run")
	}
}

func TestPluginManager_ProvidedComps_SkipsValuesWithoutCompProvider(t *testing.T) {
	game := NewGame(&GameProps{})
	game.pluginManager.track(&stubPostLoader{})

	if got := game.pluginManager.providedComps(); len(got) != 0 {
		t.Errorf("providedComps() = %v, want empty (stubPostLoader isn't a CompProvider)", got)
	}
}

type stubSaveable struct{ targets []any }

func (s *stubSaveable) SaveTargets() []any { return s.targets }

type saveTargetPayload struct{ N int }

func TestPluginManager_SaveTargets_CollectsTrackedSaveable(t *testing.T) {
	game := NewGame(&GameProps{})
	a, b := &saveTargetPayload{N: 1}, &saveTargetPayload{N: 2}
	game.pluginManager.track(&stubSaveable{targets: []any{a, b}})
	game.pluginManager.track(&stubPostLoader{})

	got := game.pluginManager.saveTargets()
	if len(got) != 2 || got[0] != any(a) || got[1] != any(b) {
		t.Errorf("saveTargets() = %v, want [%v %v]", got, a, b)
	}
}
