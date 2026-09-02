package world_test

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

func TestSpawnerFunc_Spawn_CallsUnderlyingFunc(t *testing.T) {
	wantPos := world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](7, 0), 0, 0)}
	wantVel := world.Velocity{Value: 3}
	f := world.SpawnerFunc(func(index, count int) (world.Position, world.Velocity) {
		return wantPos, wantVel
	})
	gotPos, gotVel := f.Spawn(1, 2)
	if gotPos.TopLeft.X != wantPos.TopLeft.X || gotVel.Value != wantVel.Value {
		t.Errorf("Spawn = (%+v,%+v), want (%+v,%+v)", gotPos, gotVel, wantPos, wantVel)
	}
}
