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
