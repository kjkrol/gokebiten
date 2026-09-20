package collisions

import (
	"math"

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

// impactOf is the impulse two physical sides exchange along n, from the
// velocities they carry by now, zero unless they are closing along it.
func impactOf(a, b Physics, deltaA, deltaB, n geom.Vec) float64 {
	invA, invB := inverseMass(a), inverseMass(b)
	if invA+invB == 0 {
		return 0
	}
	approach := (deltaA.X-deltaB.X)*n.X + (deltaA.Y-deltaB.Y)*n.Y
	if approach > 0 {
		return 0
	}
	return -(1 + min(a.Bounce(), b.Bounce())) * approach / (invA + invB)
}

// inverseMass is how much of an impulse a side takes — none for one nothing can move.
func inverseMass(p Physics) float64 {
	if p.Immovable() {
		return 0
	}
	return 1 / p.Weight()
}
