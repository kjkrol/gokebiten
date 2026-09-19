package main

import (
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// randomVelocity assigns a random velocity, X and Y independently drawn
// from [-rng, rng]; an X component whose magnitude falls below deadZone is
// clamped up to minSpeed so entities don't start nearly stationary.
type randomVelocity struct {
	rng, deadZone, minSpeed int32
}

func newRandomVelocity(rng, deadZone, minSpeed int32) *randomVelocity {
	return &randomVelocity{rng: rng, deadZone: deadZone, minSpeed: minSpeed}
}

func (m *randomVelocity) initialVelocity(index int) world.Velocity {
	dx := rng.Int32N(2*m.rng+1) - m.rng
	dy := rng.Int32N(2*m.rng+1) - m.rng

	if dx >= 0 && dx < m.deadZone {
		dx = m.minSpeed
	} else if dx < 0 && dx > -m.deadZone {
		dx = -m.minSpeed
	}

	var vel world.Velocity
	vel.SetDelta(geom.NewVec(float64(dx), float64(dy)))
	return vel
}
