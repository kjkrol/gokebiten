package world

import (
	"math"

	"github.com/kjkrol/gokg/geom"
)

// Velocity is an entity's current heading (Dir, a unit vector — zero when
// stationary) and speed (Value, world-units/sec), plus the sub-pixel
// remainder (AccX/AccY) MoveSystem carries between ticks.
type Velocity struct {
	Dir        geom.Vec[float64]
	Value      int32
	AccX, AccY float64
}

// Delta returns Velocity's current per-axis rate (Dir scaled by Value).
func (v Velocity) Delta() geom.Vec[int32] {
	return geom.NewVec(int32(math.Round(v.Dir.X*float64(v.Value))), int32(math.Round(v.Dir.Y*float64(v.Value))))
}

// SetDelta sets Dir/Value from a Cartesian per-axis rate — for producers/consumers that think in components, not direction+speed.
func (v *Velocity) SetDelta(d geom.Vec[int32]) {
	mag := math.Hypot(float64(d.X), float64(d.Y))
	if mag < 1e-9 {
		v.Value = 0
		return
	}
	v.Dir = geom.NewVec(float64(d.X)/mag, float64(d.Y)/mag)
	v.Value = int32(math.Round(mag))
}
