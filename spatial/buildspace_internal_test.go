package spatial

import (
	"testing"

	"github.com/kjkrol/gokebiten/physics/kinematics"
)

func TestBuildSpace_CapacityClampedHigh(t *testing.T) {
	// density=0.0025 -> raw=round(1/sqrt(0.0025))=20, clamps down to 8.
	cfg := Config{Width: 1000, Height: 1000, Toroidal: false}
	pop := Population{MaxCount: 1, MinSize: 1, MaxSize: 50}

	space := buildSpace(cfg, pop)
	if want := 8 * 8; space.BucketCapacity != want {
		t.Errorf("BucketCapacity = %d, want %d (capacity clamped to max 8)", space.BucketCapacity, want)
	}
}

func TestBuildSpace_CapacityClampedLow(t *testing.T) {
	// density=2 -> raw=round(1/sqrt(2))=1, clamps up to 2.
	cfg := Config{Width: 1000, Height: 1000, Toroidal: false}
	pop := Population{MaxCount: 200, MinSize: 1, MaxSize: 100}

	space := buildSpace(cfg, pop)
	if want := 2 * 2; space.BucketCapacity != want {
		t.Errorf("BucketCapacity = %d, want %d (capacity clamped to min 2)", space.BucketCapacity, want)
	}
}

func TestBuildSpace_CapacityUnclamped(t *testing.T) {
	// density=0.06 -> raw=round(1/sqrt(0.06))=4, within [2,8] untouched.
	cfg := Config{Width: 1000, Height: 1000, Toroidal: false}
	pop := Population{MaxCount: 6, MinSize: 1, MaxSize: 100}

	space := buildSpace(cfg, pop)
	if want := 4 * 4; space.BucketCapacity != want {
		t.Errorf("BucketCapacity = %d, want %d (capacity within range, unclamped)", space.BucketCapacity, want)
	}
}

func TestWorldModule_Reserve_PanicsOverMaxCount(t *testing.T) {
	w := &WorldModule{population: Population{MaxCount: 5}}
	w.reserve(5) // exactly at the cap, must not panic

	defer func() {
		if recover() == nil {
			t.Error("expected reserve to panic when exceeding MaxCount")
		}
	}()
	w.reserve(1)
}

func TestWorldModule_ValidateSize_PanicsOutsideBounds(t *testing.T) {
	w := &WorldModule{population: Population{MinSize: 5, MaxSize: 50}}

	// Within bounds: must not panic.
	within := kinematics.Position{}
	within.Size.X, within.Size.Y = 10, 10
	w.validateSize(1, within)

	t.Run("too small", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected validateSize to panic for a size below MinSize")
			}
		}()
		pos := kinematics.Position{}
		pos.Size.X, pos.Size.Y = 1, 10
		w.validateSize(1, pos)
	})

	t.Run("too large", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected validateSize to panic for a size above MaxSize")
			}
		}()
		pos := kinematics.Position{}
		pos.Size.X, pos.Size.Y = 10, 999
		w.validateSize(1, pos)
	})
}
