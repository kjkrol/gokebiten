package collisions

import (
	"math"

	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// normalOf is pen as a unit vector, false when there is no penetration to point along.
func normalOf(pen geom.Vec) (geom.Vec, bool) {
	mag := math.Hypot(pen.X, pen.Y)
	if mag == 0 {
		return geom.Vec{}, false
	}
	return geom.NewVec(pen.X/mag, pen.Y/mag), true
}

// impactOf is the impulse the two sides exchange along n, from the velocities
// they carry by now, zero unless they are closing along it.
func impactOf(a, b contactSide, deltaA, deltaB, n geom.Vec) float64 {
	invA, invB := inverseMass(a), inverseMass(b)
	if invA+invB == 0 {
		return 0
	}
	approach := (deltaA.X-deltaB.X)*n.X + (deltaA.Y-deltaB.Y)*n.Y
	if approach > 0 {
		return 0
	}
	return -(1 + min(a.Restitution, b.Restitution)) * approach / (invA + invB)
}

// inverseMass is how much of an impulse a side takes — zero for one that cannot move.
func inverseMass(s contactSide) float64 {
	if s.Vel == nil || math.IsInf(s.Mass, 1) {
		return 0
	}
	return 1 / s.Mass
}

// deltaOf is a side's per-axis velocity, zero when it has none to read.
func deltaOf(v *world.Velocity) geom.Vec {
	if v == nil {
		return geom.Vec{}
	}
	return v.Delta()
}
