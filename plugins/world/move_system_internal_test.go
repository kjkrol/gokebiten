package world

import "testing"

func TestClampAcc(t *testing.T) {
	// clampAcc assumes max > 0 — callers (MoveSystem.Update) only invoke it under that guard.
	cases := []struct {
		name         string
		accX, accY   float64
		max          uint32
		wantX, wantY float64
	}{
		{"magnitude within max unchanged", 3, 4, 10, 3, 4},
		{"magnitude exactly at max unchanged", 3, 4, 5, 3, 4},
		{"magnitude above max scales down preserving direction", 6, 8, 5, 3, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			accX, accY := c.accX, c.accY
			clampAcc(&accX, &accY, c.max)
			if accX != c.wantX || accY != c.wantY {
				t.Errorf("clampAcc(%v, %v, %d) = (%v, %v), want (%v, %v)",
					c.accX, c.accY, c.max, accX, accY, c.wantX, c.wantY)
			}
		})
	}
}
