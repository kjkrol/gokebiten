package collisions_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
)

func TestCollisions_LoadComps_ListsOwnedComponents(t *testing.T) {
	p := collisions.New(testSpace(t), goke.New())

	tokens := p.LoadComps()
	if len(tokens) != 4 {
		t.Fatalf("LoadComps() returned %d tokens, want 4", len(tokens))
	}
}

func TestCollisions_RegSystems_IsIdempotent(t *testing.T) {
	ecs := goke.New()
	p := collisions.New(testSpace(t), ecs)

	// Must not panic or double-register systems when called more than once —
	// RunPlan/RegSystems both guard with the same p.built check.
	p.RegSystems(ecs)
	p.RegSystems(ecs)
}
