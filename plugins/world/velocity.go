package world

import (
	"math"

	"github.com/kjkrol/gokg/geom"
)

// Velocity is an entity's current heading (Dir, a unit vector — zero when
// stationary) and speed (Value, world-units/sec).
//
// There is no sub-unit remainder to carry any more: positions are continuous,
// so an entity moving half a unit per tick moves half a unit per tick.
type Velocity struct {
	Dir   geom.Vec
	Value float64
}

// Delta returns Velocity's current per-axis rate (Dir scaled by Value).
func (v Velocity) Delta() geom.Vec {
	return geom.NewVec(v.Dir.X*v.Value, v.Dir.Y*v.Value)
}

// SetDelta sets Dir/Value from a Cartesian per-axis rate — for producers/consumers that think in components, not direction+speed.
func (v *Velocity) SetDelta(d geom.Vec) {
	mag := math.Hypot(d.X, d.Y)
	if mag < 1e-9 {
		v.Value = 0
		return
	}
	v.Dir = geom.NewVec(d.X/mag, d.Y/mag)
	v.Value = mag
}
