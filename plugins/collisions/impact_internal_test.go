package collisions

import (
	"math"
	"testing"

	"github.com/kjkrol/gokg/geom"
)

func TestImpactOf(t *testing.T) {
	elastic := func(mass float64) Physics { return Physics{Mass: mass, Restitution: 1} }
	wall := Physics{Mass: math.Inf(1), Restitution: 1}

	cases := map[string]struct {
		a, b           Physics
		deltaA, deltaB geom.Vec
		n              geom.Vec
		want           float64
	}{
		"equal masses, head-on":    {elastic(1), elastic(1), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 7},
		"already separating":       {elastic(1), elastic(1), geom.NewVec(3, 0), geom.NewVec(-4, 0), geom.NewVec(1, 0), 0},
		"across the other axis":    {elastic(1), elastic(1), geom.NewVec(0, -3), geom.NewVec(0, 4), geom.NewVec(0, 1), 7},
		"heavy into light":         {elastic(9), elastic(1), geom.NewVec(2, 0), geom.NewVec(-2, 0), geom.NewVec(-1, 0), 7.2},
		"into an immovable side":   {elastic(1), wall, geom.NewVec(3, 4), geom.Vec{}, geom.NewVec(-1, 0), 6},
		"immovable into immovable": {wall, wall, geom.Vec{}, geom.Vec{}, geom.NewVec(1, 0), 0},
		"half the bounce":          {Physics{Mass: 1, Restitution: 0.5}, elastic(1), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 5.25},
		"no bounce at all":         {Physics{Mass: 1}, elastic(1), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 3.5},
		"unnamed mass is default":  {Physics{Restitution: 1}, elastic(1), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 7},
		"restitution past its end": {Physics{Mass: 1, Restitution: 5}, elastic(1), geom.NewVec(-3, 0), geom.NewVec(4, 0), geom.NewVec(1, 0), 7},
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
