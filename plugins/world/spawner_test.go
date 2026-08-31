package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

type fakeMotion struct{ vel world.Velocity }

func (m fakeMotion) InitialVelocity(index, count int) world.Velocity { return m.vel }

func TestSpawner_Spawn_CombinesPlacementAndMotion(t *testing.T) {
	wantPos := world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](42, 0), 0, 0)}
	wantVel := world.Velocity{Dir: geom.NewVec[float64](1, 0), Value: 7}

	s := world.NewSpawner(fakePlacement{pos: wantPos}, fakeMotion{vel: wantVel})

	gotPos, gotVel := s.Spawn(1, 3)
	if gotPos.TopLeft.X != wantPos.TopLeft.X {
		t.Errorf("Spawn's Position = %+v, want %+v", gotPos, wantPos)
	}
	if gotVel.Value != wantVel.Value {
		t.Errorf("Spawn's Velocity = %+v, want %+v", gotVel, wantVel)
	}
}

func TestSpawner_Components_IsNil(t *testing.T) {
	s := world.NewSpawner(fakePlacement{}, fakeMotion{})
	if s.Components() != nil {
		t.Errorf("Components() = %v, want nil", s.Components())
	}
}

func TestSpawner_Init_IsNoop(t *testing.T) {
	s := world.NewSpawner(fakePlacement{}, fakeMotion{})
	s.Init(nil, 0, 0, uid.UID64(0))
}
