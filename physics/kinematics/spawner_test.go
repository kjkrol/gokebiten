package kinematics_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/uid"
)

type fakePlacement struct{ pos kinematics.Position }

func (p fakePlacement) Place(index, count int) kinematics.Position { return p.pos }
func (p fakePlacement) Components() []goke.Addable                 { return nil }
func (p fakePlacement) Init(*goke.Cursor, int, int, uid.UID64)     {}

type fakeMotion struct{ vel kinematics.Velocity }

func (m fakeMotion) InitialVelocity(index, count int) kinematics.Velocity { return m.vel }

func TestSpawner_Spawn_CombinesPlacementAndMotion(t *testing.T) {
	wantPos := kinematics.Position{AccX: 42}
	wantVel := kinematics.Velocity{}
	wantVel.Vec.X = 7

	s := kinematics.NewSpawner(fakePlacement{pos: wantPos}, fakeMotion{vel: wantVel})

	gotPos, gotVel := s.Spawn(1, 3)
	if gotPos.AccX != wantPos.AccX {
		t.Errorf("Spawn's Position = %+v, want %+v", gotPos, wantPos)
	}
	if gotVel.Vec.X != wantVel.Vec.X {
		t.Errorf("Spawn's Velocity = %+v, want %+v", gotVel, wantVel)
	}
}

func TestSpawner_Components_IsNil(t *testing.T) {
	s := kinematics.NewSpawner(fakePlacement{}, fakeMotion{})
	if s.Components() != nil {
		t.Errorf("Components() = %v, want nil", s.Components())
	}
}

func TestSpawner_Init_IsNoop(t *testing.T) {
	s := kinematics.NewSpawner(fakePlacement{}, fakeMotion{})
	// Must not panic with nil/zero arguments.
	s.Init(nil, 0, 0, uid.UID64(0))
}
