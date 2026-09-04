package collisions_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"

	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	gokgspatial "github.com/kjkrol/gokg/spatial"
	"github.com/kjkrol/uid"
)

func testSpace(t *testing.T) *gokg.Space {
	t.Helper()
	space, err := gokg.NewSpace(gokg.Config{
		Width: 1000, Height: 1000,
		BucketSize: gokgspatial.ResolutionFrom(64), BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("gokg.NewSpace: %v", err)
	}
	return space
}

func posAt(x, y, w, h uint32) world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec(x, y), w, h)}
}

// seedBroadPhaseEntity spawns one Position+Velocity+Collision entity via si
// (must be called from within an ecs.Setup OnInit) and inserts it into
// space's spatial index — BroadPhase.Update discovers neighbors purely
// through that index, not through goke's Query.
func seedBroadPhaseEntity(t *testing.T, si *goke.SysInit, space *gokg.Space, pos world.Position) uid.UID64 {
	t.Helper()
	var posComp goke.Comp[world.Position]
	var velComp goke.Comp[world.Velocity]
	var collComp goke.Comp[collisions.Collision]
	f := si.NewFactory(&posComp, &velComp, &collComp)
	f.Create(1)
	f.Next()
	posComp.Slice(&f.Cursor)[0] = pos
	id := f.IDs[0]
	space.Insert(id, pos.AABB)
	return id
}

// seedNonCollidableEntity is seedBroadPhaseEntity without a Collision
// component — a world entity present in the shared spatial index that
// should never be treated as a collision candidate.
func seedNonCollidableEntity(t *testing.T, si *goke.SysInit, space *gokg.Space, pos world.Position) uid.UID64 {
	t.Helper()
	var posComp goke.Comp[world.Position]
	var velComp goke.Comp[world.Velocity]
	f := si.NewFactory(&posComp, &velComp)
	f.Create(1)
	f.Next()
	posComp.Slice(&f.Cursor)[0] = pos
	id := f.IDs[0]
	space.Insert(id, pos.AABB)
	return id
}

// seedMovingCollidableEntity is seedBroadPhaseEntity plus a starting Velocity, for driving it through world.MoveSystem.
func seedMovingCollidableEntity(t *testing.T, si *goke.SysInit, space *gokg.Space, pos world.Position, vel world.Velocity) uid.UID64 {
	t.Helper()
	var posComp goke.Comp[world.Position]
	var velComp goke.Comp[world.Velocity]
	var collComp goke.Comp[collisions.Collision]
	f := si.NewFactory(&posComp, &velComp, &collComp)
	f.Create(1)
	f.Next()
	posComp.Slice(&f.Cursor)[0] = pos
	velComp.Slice(&f.Cursor)[0] = vel
	id := f.IDs[0]
	space.Insert(id, pos.AABB)
	return id
}

func hasHit(q *goke.Query) map[uid.UID64]bool {
	found := map[uid.UID64]bool{}
	q.All()
	for q.Next() {
		for _, id := range q.Cursor().IDs {
			found[id] = true
		}
	}
	return found
}

func TestBroadPhase_Update_DetectsOverlappingNeighbors(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var hitQ *goke.Query
	var hitTag goke.Comp[collisions.Hit]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		idB = seedBroadPhaseEntity(t, si, space, posAt(5, 0, 10, 10)) // overlaps A by 5px
		space.Flush(nil)
		hitQ = si.NewQueryBuilder(&hitTag).Build()
	}})

	bp := collisions.NewBroadPhase(space)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	got := hasHit(hitQ)
	if !got[idA] || !got[idB] {
		t.Errorf("expected both overlapping entities to get a Hit tag, got %v (idA=%v idB=%v)", got, idA, idB)
	}
}

func TestBroadPhase_Update_NoOverlap_NoHitAdded(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var hitQ *goke.Query
	var hitTag goke.Comp[collisions.Hit]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		idB = seedBroadPhaseEntity(t, si, space, posAt(900, 900, 10, 10)) // far away
		space.Flush(nil)
		hitQ = si.NewQueryBuilder(&hitTag).Build()
	}})

	bp := collisions.NewBroadPhase(space)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	got := hasHit(hitQ)
	if got[idA] || got[idB] {
		t.Errorf("expected neither distant entity to get a Hit tag, got %v", got)
	}
}

func TestBroadPhase_Update_SingleEntity_SelfExcluded(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var id uid.UID64
	var hitQ *goke.Query
	var hitTag goke.Comp[collisions.Hit]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		id = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		space.Flush(nil)
		hitQ = si.NewQueryBuilder(&hitTag).Build()
	}})

	bp := collisions.NewBroadPhase(space)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	if got := hasHit(hitQ); got[id] {
		t.Errorf("expected a lone entity to never touch itself, got %v", got)
	}
}

// TestBroadPhase_Update_IgnoresNonCollidableNeighbor guards the filter added
// once world's shared Space started indexing every entity, not just
// collidable ones — a geometric neighbor without Collision must never be
// recorded as touching.
func TestBroadPhase_Update_IgnoresNonCollidableNeighbor(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var collQ *goke.Query
	var collTag goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		idB = seedNonCollidableEntity(t, si, space, posAt(5, 0, 10, 10)) // overlaps A, no Collision
		space.Flush(nil)
		collQ = si.NewQueryBuilder(&collTag).Build()
	}})

	bp := collisions.NewBroadPhase(space)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	collQ.All()
	for collQ.Next() {
		cur := collQ.Cursor()
		tags := collTag.Slice(cur)
		for i, id := range cur.IDs {
			if id != idA {
				continue
			}
			for ti := uint8(0); ti < tags[i].TouchingCount; ti++ {
				if tags[i].Touching[ti] == idB {
					t.Errorf("entityA recorded idB (%v) as touching — idB has no Collision and should have been filtered out", idB)
				}
			}
		}
	}
}

// TestBroadPhase_Update_DetectsEntityMovedByMoveSystem is the regression
// test for the bug where world.MoveSystem and collisions each owned a
// separate spatial index: an entity moved purely by MoveSystem (no
// collision involved yet) must still be found by BroadPhase once it
// overlaps a neighbor, because they now share one Space.
func TestBroadPhase_Update_DetectsEntityMovedByMoveSystem(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var hitQ *goke.Query
	var hitTag goke.Comp[collisions.Hit]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedMovingCollidableEntity(t, si, space, posAt(0, 0, 10, 10), world.Velocity{Dir: geom.NewVec[float64](1, 0), Value: 105})
		idB = seedBroadPhaseEntity(t, si, space, posAt(100, 0, 10, 10)) // far from idA's start, not yet overlapping
		space.Flush(nil)
		hitQ = si.NewQueryBuilder(&hitTag).Build()
	}})

	moveSystem := world.NewMoveSystem(space, 0)
	moveHandle := ecs.RegSys(moveSystem)
	bp := collisions.NewBroadPhase(space)
	bpHandle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(moveHandle, d)
		ctx.Run(bpHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second) // idA moves 105 units, landing 5px into idB

	got := hasHit(hitQ)
	if !got[idA] || !got[idB] {
		t.Errorf("expected BroadPhase to detect idA after MoveSystem moved it into idB, got %v (idA=%v idB=%v)", got, idA, idB)
	}
}
