package vision_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/vision"
)

// Every component vision owns has to be listed, or a save that contains one
// fails to load with "no matching LoadComp was provided".
func TestModule_DeclaresEveryComponentItOwns(t *testing.T) {
	p := vision.NewPlugin(testWorldPlugin())
	ctx := &installCtx{ecs: goke.New()}
	if err := p.Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	got := len(goke.ProvidedComps(ctx.tracked...))
	if got != 3 {
		t.Errorf("vision declares %d components, want 3 (Sight, Sighted, SightOutline)", got)
	}
}

// RegSystems runs again whenever a Stage is re-entered; it must not stack up a
// second copy of the scan.
func TestModule_RegSystemsIsIdempotent(t *testing.T) {
	p := vision.NewPlugin(testWorldPlugin())
	ctx := &installCtx{ecs: goke.New()}
	if err := p.Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, m := range ctx.tracked {
		mod := m.(goke.Module)
		mod.RegSystems(ctx.ecs)
		mod.RegSystems(ctx.ecs)
	}
}
