package collisions_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
)

func TestCollisions_LoadComps_ListsOwnedComponents(t *testing.T) {
	p := collisions.New(testSpace(t), goke.New(), 0)

	tokens := p.LoadComps()
	if len(tokens) != 5 {
		t.Fatalf("LoadComps() returned %d tokens, want 5", len(tokens))
	}
}

func TestCollisions_RegSystems_IsIdempotent(t *testing.T) {
	ecs := goke.New()
	p := collisions.New(testSpace(t), ecs, 0)

	// Must not panic or double-register systems when called more than once —
	// RegSystems guards with p.built.
	p.RegSystems(ecs)
	p.RegSystems(ecs)
}
