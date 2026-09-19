package main

import (
	"testing"
)

func TestRandomVelocity_InitialVelocity_XWithinRangeAndDeadZoned(t *testing.T) {
	const rng, deadZone, minSpeed = int32(50), int32(10), int32(20)
	// Delta() round-trips what was drawn now: Dir+Value no longer rounds the
	// magnitude to a whole number on the way through Hypot/normalize, so the
	// unit of slack this test used to allow is gone.
	const tolerance = 1e-9
	m := newRandomVelocity(rng, deadZone, minSpeed)

	for i := range 500 {
		d := m.initialVelocity(i).Delta()

		if d.X < float64(-rng)-tolerance || d.X > float64(rng)+tolerance {
			t.Fatalf("X = %v, want within [-%d,%d] (+/-%v)", d.X, rng, rng, tolerance)
		}
		if d.X >= 0 && d.X < float64(deadZone)-tolerance && d.X != float64(minSpeed) {
			t.Errorf("X = %v is inside the positive dead zone [0,%d) but wasn't clamped to MinSpeed=%d", d.X, deadZone, minSpeed)
		}
		if d.X < 0 && d.X > float64(-deadZone)+tolerance && d.X != float64(-minSpeed) {
			t.Errorf("X = %v is inside the negative dead zone (-%d,0) but wasn't clamped to -MinSpeed=%d", d.X, deadZone, -minSpeed)
		}

		if d.Y < float64(-rng)-tolerance || d.Y > float64(rng)+tolerance {
			t.Errorf("Y = %v, want within [-%d,%d] (+/-%v)", d.Y, rng, rng, tolerance)
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
		d := m.initialVelocity(i).Delta()
		if d.Y != float64(minSpeed) && d.Y != float64(-minSpeed) {
			sawUnclampedY = true
			break
		}
	}
	if !sawUnclampedY {
		t.Error("expected at least one Y draw not equal to ±MinSpeed across 2000 samples — Y should never be dead-zone clamped")
	}
}
