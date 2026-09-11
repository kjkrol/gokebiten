package engine

import (
	"testing"

	"github.com/kjkrol/goke/v3"
)

type stubPostLoader struct{ ran bool }

func (s *stubPostLoader) SetupSystems() []goke.System { return nil }
func (s *stubPostLoader) PostLoad() goke.System {
	return goke.SystemFn{OnInit: func(si *goke.SysInit) { s.ran = true }}
}

func TestEngine_PostLoadSystems_RunsTrackedPostLoader(t *testing.T) {
	engine := newTestEngine(nil)
	stub := &stubPostLoader{}
	engine.track(stub)

	systems := engine.postLoadSystems()
	if len(systems) != 1 {
		t.Fatalf("postLoadSystems() returned %d systems, want 1", len(systems))
	}

	goke.New().Setup(systems...)
	if !stub.ran {
		t.Error("expected the tracked PostLoader's system to have run")
	}
}

func TestEngine_ProvidedComps_SkipsValuesWithoutCompProvider(t *testing.T) {
	engine := newTestEngine(nil)
	engine.track(&stubPostLoader{})

	if got := engine.providedComps(); len(got) != 0 {
		t.Errorf("providedComps() = %v, want empty (stubPostLoader isn't a CompProvider)", got)
	}
}

type stubSerializable struct{ targets []any }

func (s *stubSerializable) Persisted() []any { return s.targets }

type saveTargetPayload struct{ N int }

func TestEngine_SaveTargets_CollectsTrackedSerializable(t *testing.T) {
	engine := newTestEngine(nil)
	a, b := &saveTargetPayload{N: 1}, &saveTargetPayload{N: 2}
	engine.track(&stubSerializable{targets: []any{a, b}})
	engine.track(&stubPostLoader{})

	got := engine.saveTargets()
	if len(got) != 1 {
		t.Fatalf("saveTargets() returned %d groups, want 1", len(got))
	}
	for _, targets := range got {
		if len(targets) != 2 || targets[0] != any(a) || targets[1] != any(b) {
			t.Errorf("saveTargets() targets = %v, want [%v %v]", targets, a, b)
		}
	}
}
