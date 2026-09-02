package main

import (
	"testing"
)

func TestRandomVelocity_InitialVelocity_XWithinRangeAndDeadZoned(t *testing.T) {
	const rng, deadZone, minSpeed = int32(50), int32(10), int32(20)
	// Velocity now stores Dir+Value, not raw components — recovering a
	// component via Delta() can be off by 1 from what was drawn, since it
	// round-trips through Hypot/normalize.
	const tolerance = int32(1)
	m := newRandomVelocity(rng, deadZone, minSpeed)

	for i := range 500 {
		d := m.initialVelocity(i, 500).Delta()

		if d.X < -rng-tolerance || d.X > rng+tolerance {
			t.Fatalf("X = %d, want within [-%d,%d] (+/-%d for Dir/Value rounding)", d.X, rng, rng, tolerance)
		}
		if d.X >= 0 && d.X < deadZone-tolerance && d.X != minSpeed {
			t.Errorf("X = %d is inside the positive dead zone [0,%d) but wasn't clamped to MinSpeed=%d", d.X, deadZone, minSpeed)
		}
		if d.X < 0 && d.X > -deadZone+tolerance && d.X != -minSpeed {
			t.Errorf("X = %d is inside the negative dead zone (-%d,0) but wasn't clamped to -MinSpeed=%d", d.X, deadZone, -minSpeed)
		}

		if d.Y < -rng-tolerance || d.Y > rng+tolerance {
			t.Errorf("Y = %d, want within [-%d,%d] (+/-%d for Dir/Value rounding)", d.Y, rng, rng, tolerance)
		}
	}
}

func TestRandomVelocity_InitialVelocity_YHasNoDeadZone(t *testing.T) {
	// Y is drawn from [-Range,Range] with no dead-zone clamp at all (only X
	// gets one) — with DeadZone==Range every non-zero-magnitude X draw would
	// get clamped to MinSpeed, but Y must still be free to land inside
	// (-DeadZone,DeadZone), including exactly 0, unclamped.
	const rng, deadZone, minSpeed = int32(20), int32(20), int32(5)
	m := newRandomVelocity(rng, deadZone, minSpeed)

	sawUnclampedY := false
	for i := range 2000 {
		d := m.initialVelocity(i, 2000).Delta()
		if d.Y != minSpeed && d.Y != -minSpeed {
			sawUnclampedY = true
			break
		}
	}
	if !sawUnclampedY {
		t.Error("expected at least one Y draw not equal to ±MinSpeed across 2000 samples — Y should never be dead-zone clamped")
	}
}
