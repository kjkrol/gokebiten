package stats_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/stats"
)

func TestHandler_OnCollision_IncrementsCounter(t *testing.T) {
	s := &stats.Stats{}
	h := stats.NewHandler(s)

	h.OnCollision(nil, collisions.CollisionEvent{})
	if s.Counter != 1 {
		t.Fatalf("Counter = %d, want 1", s.Counter)
	}

	h.OnCollision(nil, collisions.CollisionEvent{})
	if s.Counter != 2 {
		t.Errorf("Counter = %d, want 2", s.Counter)
	}
}
