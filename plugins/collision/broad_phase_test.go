package collision_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collision"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/uid"
)

func testSpace(t *testing.T) *aabbworld.Space {
	t.Helper()
	space, err := aabbworld.NewSpace(aabbworld.Config{
		Width: 1000, Height: 1000,
		BucketSize: 64, BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("aabbworld.NewSpace: %v", err)
	}
	return space
}

func posAt(x, y, w, h float64) world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec(x, y), w, h)}
}

// seedBroadPhaseEntity spawns one entity carrying Collider and inserts it into space's index.
func seedBroadPhaseEntity(t *testing.T, si *goke.SysInit, space *aabbworld.Space, pos world.Position) uid.UID64 {
	t.Helper()
	var baseComp goke.Comp[world.Base]
	var collComp goke.Comp[collision.Collider]
	f := si.NewFactory(&baseComp, &collComp)
	f.Create(1)
	f.Next()
	baseComp.Slice(&f.Cursor)[0].Pos = pos
	id := f.IDs[0]
	space.Insert(id, &pos.AABB)
	return id
}

// seedNonCollidableEntity is seedBroadPhaseEntity without a Collider.
func seedNonCollidableEntity(t *testing.T, si *goke.SysInit, space *aabbworld.Space, pos world.Position) uid.UID64 {
	t.Helper()
	var baseComp goke.Comp[world.Base]
	f := si.NewFactory(&baseComp)
	f.Create(1)
	f.Next()
	baseComp.Slice(&f.Cursor)[0].Pos = pos
	id := f.IDs[0]
	space.Insert(id, &pos.AABB)
	return id
}

// seedMovingCollidableEntity is seedBroadPhaseEntity plus a starting Velocity.
func seedMovingCollidableEntity(t *testing.T, si *goke.SysInit, space *aabbworld.Space, pos world.Position, vel world.Velocity) uid.UID64 {
	t.Helper()
	var baseComp goke.Comp[world.Base]
	var collComp goke.Comp[collision.Collider]
	f := si.NewFactory(&baseComp, &collComp)
	f.Create(1)
	f.Next()
	baseComp.Slice(&f.Cursor)[0] = world.Base{Pos: pos, Vel: vel}
	id := f.IDs[0]
	space.Insert(id, &pos.AABB)
	return id
}

// broadTick seeds a world, runs ticks of movement and broad phase, and returns the last pairs.
func broadTick(t *testing.T, ticks int, seed func(si *goke.SysInit, space *aabbworld.Space)) []collision.Candidate {
	t.Helper()
	space := testSpace(t)
	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		seed(si, space)
		space.Flush(nil)
	}})

	var found collision.Candidates
	move := ecs.RegSys(world.NewMoveSystem(space))
	broad := ecs.RegSys(collision.NewBroadPhase(space, &found))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(move, d)
		ctx.Run(broad, d)
		ctx.Sync()
	})
	for range ticks {
		ecs.Tick(time.Second)
	}
	return found.All()
}

// paired reports whether found names a and b as a pair, either way round.
func paired(found []collision.Candidate, a, b uid.UID64) bool {
	for _, c := range found {
		if (c.A == a && c.B == b) || (c.A == b && c.B == a) {
			return true
		}
	}
	return false
}

func TestBroadPhase_Update_NamesOverlappingNeighborsOnce(t *testing.T) {
	var idA, idB uid.UID64
	found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		idB = seedBroadPhaseEntity(t, si, space, posAt(5, 0, 10, 10))
	})

	if len(found) != 1 || !paired(found, idA, idB) {
		t.Errorf("found %v, want the one pair (%v, %v)", found, idA, idB)
	}
}

func TestBroadPhase_Update_StartsTheListAfreshEachTick(t *testing.T) {
	found := broadTick(t, 3, func(si *goke.SysInit, space *aabbworld.Space) {
		seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		seedBroadPhaseEntity(t, si, space, posAt(5, 0, 10, 10))
	})

	if len(found) != 1 {
		t.Errorf("%d pairs after three ticks over one overlapping pair, want 1", len(found))
	}
}

func TestBroadPhase_Update_FarApart_NothingNamed(t *testing.T) {
	found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
		seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		seedBroadPhaseEntity(t, si, space, posAt(900, 900, 10, 10))
	})

	if len(found) != 0 {
		t.Errorf("found %v between two entities a world apart, want nothing", found)
	}
}

func TestBroadPhase_Update_SingleEntity_NeverPairsWithItself(t *testing.T) {
	found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
		seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
	})

	if len(found) != 0 {
		t.Errorf("found %v for a lone entity, want nothing", found)
	}
}

func TestBroadPhase_Update_IgnoresNonCollidableNeighbor(t *testing.T) {
	found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
		seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		seedNonCollidableEntity(t, si, space, posAt(5, 0, 10, 10))
	})

	if len(found) != 0 {
		t.Errorf("found %v, want nothing — the neighbor cannot collide", found)
	}
}

func TestBroadPhase_Update_FollowsAnEntityMovedByMoveSystem(t *testing.T) {
	var idA, idB uid.UID64
	found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
		idA = seedMovingCollidableEntity(t, si, space, posAt(78, 0, 10, 10), world.Velocity{Dir: geom.NewVec(1, 0), Value: 1000})
		idB = seedBroadPhaseEntity(t, si, space, posAt(100, 0, 10, 10))
	})

	if !paired(found, idA, idB) {
		t.Errorf("found %v, want (%v, %v) once MoveSystem brought A within reach", found, idA, idB)
	}
}

func TestBroadPhase_ReachIsOneTickOfClosing(t *testing.T) {
	const size = 10
	for gap, want := range map[float64]bool{size: true, size + 1: false} {
		var idA, idB uid.UID64
		found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
			idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, size, size))
			idB = seedBroadPhaseEntity(t, si, space, posAt(size+gap, 0, size, size))
		})

		if got := paired(found, idA, idB); got != want {
			t.Errorf("two %v-unit entities %v apart: paired = %v, want %v", size, gap, got, want)
		}
	}
}

func TestBroadPhase_ReachFollowsEachEntitysOwnSize(t *testing.T) {
	var small, big, otherSmall uid.UID64
	found := broadTick(t, 1, func(si *goke.SysInit, space *aabbworld.Space) {
		small = seedBroadPhaseEntity(t, si, space, posAt(100, 300, 2, 2))
		big = seedBroadPhaseEntity(t, si, space, posAt(142, 250, 100, 100))
		otherSmall = seedBroadPhaseEntity(t, si, space, posAt(58, 300, 2, 2))
	})

	if !paired(found, small, big) {
		t.Errorf("found %v — a 100-unit entity 40 away reaches 50, so it and the small one are a pair", found)
	}
	if paired(found, small, otherSmall) {
		t.Errorf("found %v — two 2-unit entities 40 apart reach 1 each, and are no pair", found)
	}
}
