package randomvelocity_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/randomvelocity"
)

func TestRandomVelocity_InitialVelocity_XWithinRangeAndDeadZoned(t *testing.T) {
	const rng, deadZone, minSpeed = int32(50), int32(10), int32(20)
	m := randomvelocity.New(rng, deadZone, minSpeed)

	for i := range 500 {
		vel := m.InitialVelocity(i, 500)

		if vel.Vec.X < -rng || vel.Vec.X > rng {
			t.Fatalf("X = %d, want within [-%d,%d]", vel.Vec.X, rng, rng)
		}
		if vel.Vec.X >= 0 && vel.Vec.X < deadZone && vel.Vec.X != minSpeed {
			t.Errorf("X = %d is inside the positive dead zone [0,%d) but wasn't clamped to MinSpeed=%d", vel.Vec.X, deadZone, minSpeed)
		}
		if vel.Vec.X < 0 && vel.Vec.X > -deadZone && vel.Vec.X != -minSpeed {
			t.Errorf("X = %d is inside the negative dead zone (-%d,0) but wasn't clamped to -MinSpeed=%d", vel.Vec.X, deadZone, -minSpeed)
		}

		if vel.Vec.Y < -rng || vel.Vec.Y > rng {
			t.Errorf("Y = %d, want within [-%d,%d]", vel.Vec.Y, rng, rng)
		}
	}
}

func TestRandomVelocity_InitialVelocity_YHasNoDeadZone(t *testing.T) {
	// Y is drawn from [-Range,Range] with no dead-zone clamp at all (only X
	// gets one) — with DeadZone==Range every non-zero-magnitude X draw would
	// get clamped to MinSpeed, but Y must still be free to land inside
	// (-DeadZone,DeadZone), including exactly 0, unclamped.
	const rng, deadZone, minSpeed = int32(20), int32(20), int32(5)
	m := randomvelocity.New(rng, deadZone, minSpeed)

	sawUnclampedY := false
	for i := range 2000 {
		vel := m.InitialVelocity(i, 2000)
		if vel.Vec.Y != minSpeed && vel.Vec.Y != -minSpeed {
			sawUnclampedY = true
			break
		}
	}
	if !sawUnclampedY {
		t.Error("expected at least one Y draw not equal to ±MinSpeed across 2000 samples — Y should never be dead-zone clamped")
	}
}
