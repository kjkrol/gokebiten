package collisions

import (
	"math"
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

// movingSide is a side that can take an impulse; its velocity is passed to
// impactOf separately, the way the narrow phase passes what the tick left it.
func movingSide(mass, restitution float64) contactSide {
	return contactSide{Vel: &world.Velocity{}, Mass: mass, Restitution: restitution}
}

// immovableSide is what a Static entity resolves to: no Velocity to read and infinite mass.
func immovableSide() contactSide {
	return contactSide{Mass: math.Inf(1), Restitution: FullRestitution}
}

func TestImpactOf(t *testing.T) {
	const full = FullRestitution
	cases := map[string]struct {
		a, b           contactSide
		deltaA, deltaB geom.Vec
		n              geom.Vec
		want           float64
	}{
		"equal masses, head-on":    {movingSide(1, full), movingSide(1, full), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 7},
		"already separating":       {movingSide(1, full), movingSide(1, full), geom.NewVec(3, 0), geom.NewVec(-4, 0), geom.NewVec(1, 0), 0},
		"across the other axis":    {movingSide(1, full), movingSide(1, full), geom.NewVec(0, -3), geom.NewVec(0, 4), geom.NewVec(0, 1), 7},
		"heavy into light":         {movingSide(9, full), movingSide(1, full), geom.NewVec(2, 0), geom.NewVec(-2, 0), geom.NewVec(-1, 0), 7.2},
		"into an immovable side":   {movingSide(1, full), immovableSide(), geom.NewVec(3, 4), geom.Vec{}, geom.NewVec(-1, 0), 6},
		"immovable into immovable": {immovableSide(), immovableSide(), geom.Vec{}, geom.Vec{}, geom.NewVec(1, 0), 0},
		"half the bounce":          {movingSide(1, 0.5), movingSide(1, full), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 5.25},
		"no bounce at all":         {movingSide(1, 0), movingSide(1, full), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 3.5},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := impactOf(c.a, c.b, c.deltaA, c.deltaB, c.n); math.Abs(got-c.want) > 1e-9 {
				t.Errorf("impactOf = %v, want %v", got, c.want)
			}
		})
	}
}

func TestNormalOf_ZeroPenetrationHasNoDirection(t *testing.T) {
	if _, ok := normalOf(geom.Vec{}); ok {
		t.Error("normalOf(zero) reported a direction")
	}
	if n, ok := normalOf(geom.NewVec(-5, 0)); !ok || n != geom.NewVec(-1, 0) {
		t.Errorf("normalOf({-5,0}) = %v (ok %v), want {-1,0}", n, ok)
	}
}
