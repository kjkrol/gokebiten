package physics_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics"
	"github.com/kjkrol/gokebiten/spatial"
)

func TestPhysics_LoadComps_ListsOwnedComponents(t *testing.T) {
	space := spatial.NewWorldModule(
		spatial.Config{Width: 100, Height: 100},
		spatial.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	).Space()
	p := physics.New(space, goke.New(), 10, 0)

	tokens := p.LoadComps()
	if len(tokens) != 6 {
		t.Fatalf("LoadComps() returned %d tokens, want 6", len(tokens))
	}
}

func TestPhysics_RegSystems_IsIdempotent(t *testing.T) {
	space := spatial.NewWorldModule(
		spatial.Config{Width: 100, Height: 100},
		spatial.Population{MaxCount: 1, MinSize: 1, MaxSize: 10},
	).Space()
	ecs := goke.New()
	p := physics.New(space, ecs, 10, 0)

	// Must not panic or double-register systems when called more than once —
	// RunPlan/RegSystems both guard with the same p.built check.
	p.RegSystems(ecs)
	p.RegSystems(ecs)
}
