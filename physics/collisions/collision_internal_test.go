package collisions

import (
	"testing"

	"github.com/kjkrol/uid"
)

func TestCollision_AddTouching_Dedup(t *testing.T) {
	var c Collision
	id := uid.UID64(42)

	c.addTouching(id)
	c.addTouching(id)
	c.addTouching(id)

	if c.TouchingCount != 1 {
		t.Fatalf("TouchingCount = %d, want 1 after adding the same id three times", c.TouchingCount)
	}
	if c.Touching[0] != id {
		t.Errorf("Touching[0] = %v, want %v", c.Touching[0], id)
	}
}

func TestCollision_AddTouching_CapsAtMaxTouching(t *testing.T) {
	var c Collision
	for i := range MaxTouching + 5 {
		c.addTouching(uid.UID64(i))
	}

	if c.TouchingCount != MaxTouching {
		t.Fatalf("TouchingCount = %d, want %d (extras past the cap silently dropped)", c.TouchingCount, MaxTouching)
	}
	for i := range MaxTouching {
		if c.Touching[i] != uid.UID64(i) {
			t.Errorf("Touching[%d] = %v, want %v", i, c.Touching[i], uid.UID64(i))
		}
	}
}

func TestCollision_Clear(t *testing.T) {
	var c Collision
	c.addTouching(uid.UID64(1))
	c.addTouching(uid.UID64(2))

	c.clear()

	if c.TouchingCount != 0 {
		t.Errorf("TouchingCount = %d, want 0 after clear", c.TouchingCount)
	}
	// clear only resets the count, not the backing array — a stale entry
	// there must never be observed since TouchingCount gates all reads.
	c.addTouching(uid.UID64(3))
	if c.TouchingCount != 1 || c.Touching[0] != uid.UID64(3) {
		t.Errorf("expected a fresh add after clear to start at index 0 with the new id, got count=%d Touching[0]=%v", c.TouchingCount, c.Touching[0])
	}
}
