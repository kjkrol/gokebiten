package elastic

import (
	"testing"

	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokg/geom"
)

func vel(x, y int32) *kinematics.Velocity {
	return &kinematics.Velocity{Vec: geom.NewVec(x, y)}
}

func TestSwapVelocity(t *testing.T) {
	cases := []struct {
		name           string
		penX, penY     int32
		aX, aY, bX, bY int32
		wantSwap       bool
	}{
		{"pen.X>0, approaching (relVelX<0) -> swap X", 5, 0, -3, 0, 4, 0, true},
		{"pen.X>0, already separating (relVelX>0) -> no swap", 5, 0, 3, 0, -4, 0, false},
		{"pen.X<0, approaching (relVelX>0) -> swap X", -5, 0, 3, 0, -4, 0, true},
		{"pen.X<0, already separating (relVelX<0) -> no swap", -5, 0, -3, 0, 4, 0, false},
		{"pen.X==0, pen.Y>0, approaching -> swap Y", 0, 5, 0, -3, 0, 4, true},
		{"pen.X==0, pen.Y>0, already separating -> no swap", 0, 5, 0, 3, 0, -4, false},
		{"pen.X==0, pen.Y==0 -> no-op", 0, 0, 1, 2, 3, 4, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			velA := vel(c.aX, c.aY)
			velB := vel(c.bX, c.bY)
			pen := geom.Vec[int32]{X: c.penX, Y: c.penY}

			swapVelocity(velA, velB, pen)

			swapped := velA.Vec.X != c.aX || velA.Vec.Y != c.aY || velB.Vec.X != c.bX || velB.Vec.Y != c.bY
			if swapped != c.wantSwap {
				t.Errorf("swapVelocity: velA=%+v velB=%+v, wantSwap=%v", velA.Vec, velB.Vec, c.wantSwap)
			}
			if c.wantSwap {
				if c.penX != 0 {
					if velA.Vec.X != c.bX || velB.Vec.X != c.aX {
						t.Errorf("expected X components swapped, got velA.X=%d velB.X=%d", velA.Vec.X, velB.Vec.X)
					}
				} else {
					if velA.Vec.Y != c.bY || velB.Vec.Y != c.aY {
						t.Errorf("expected Y components swapped, got velA.Y=%d velB.Y=%d", velA.Vec.Y, velB.Vec.Y)
					}
				}
			}
		})
	}
}

func TestHandler_OnCollision_NilVelocitySkipsSwap(t *testing.T) {
	h := NewHandler()

	// Must not panic when one side is immovable (nil Velocity).
	h.OnCollision(nil, collisions.CollisionEvent{VelA: nil, VelB: vel(1, 0), Penetration: geom.Vec[int32]{X: 1}})
	h.OnCollision(nil, collisions.CollisionEvent{VelA: vel(1, 0), VelB: nil, Penetration: geom.Vec[int32]{X: 1}})
}

func TestHandler_OnCollision_SwapsOnRealContact(t *testing.T) {
	h := NewHandler()
	velA := vel(-3, 0)
	velB := vel(4, 0)

	h.OnCollision(nil, collisions.CollisionEvent{VelA: velA, VelB: velB, Penetration: geom.Vec[int32]{X: 5}})

	if velA.Vec.X != 4 || velB.Vec.X != -3 {
		t.Errorf("velA=%+v velB=%+v, want swapped X components", velA.Vec, velB.Vec)
	}
}
