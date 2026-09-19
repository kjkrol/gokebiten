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

func posAt(x, y, w, h float64) world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec(x, y), w, h)}
}

// seedBroadPhaseEntity spawns one Position+Velocity+Collision entity via si
// (must be called from within an ecs.Setup OnInit) and inserts it into
// space's spatial index — BroadPhase.Update discovers neighbors purely
// through that index, not through goke's Query.
//
// Setting CanCollide by hand is what collisions.Collidable does at a real
// spawn; these fixtures build entities without going through an EntKind, so
// they have to do the template's half themselves.
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
	space.SetCapabilities(id, collisions.CanCollide)
	return id
}

// seedNonCollidableEntity is seedBroadPhaseEntity without a Collision
// component and without CanCollide — a world entity present in the shared
// spatial index that should never be treated as a collision candidate.
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
	space.SetCapabilities(id, collisions.CanCollide)
	return id
}

// recorded reports which entities q matched came out of the tick with at
// least one candidate to their name.
func recorded(q *goke.Query, comp goke.Comp[collisions.Collision]) map[uid.UID64]bool {
	found := map[uid.UID64]bool{}
	q.All()
	for q.Next() {
		cur := q.Cursor()
		slice := comp.Slice(cur)
		for i, id := range cur.IDs {
			if slice[i].TouchingCount > 0 {
				found[id] = true
			}
		}
	}
	return found
}

// testProbeMargin is what world would hand the broad phase for the 10-unit
// entities these tests use: 2*MaxStep, MaxStep being MinSize/2.
const testProbeMargin = 10

func TestBroadPhase_Update_DetectsOverlappingNeighbors(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var candidateQ *goke.Query
	var candidates goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		idB = seedBroadPhaseEntity(t, si, space, posAt(5, 0, 10, 10)) // overlaps A by 5px
		space.Flush(nil)
		candidateQ = si.NewQueryBuilder(&candidates).Build()
	}})

	bp := collisions.NewBroadPhase(space, testProbeMargin)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	got := recorded(candidateQ, candidates)
	if !got[idA] || !got[idB] {
		t.Errorf("expected both overlapping entities to record a candidate, got %v (idA=%v idB=%v)", got, idA, idB)
	}
}

func TestBroadPhase_Update_NoOverlap_NothingRecorded(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var candidateQ *goke.Query
	var candidates goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		idB = seedBroadPhaseEntity(t, si, space, posAt(900, 900, 10, 10)) // far away
		space.Flush(nil)
		candidateQ = si.NewQueryBuilder(&candidates).Build()
	}})

	bp := collisions.NewBroadPhase(space, testProbeMargin)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	got := recorded(candidateQ, candidates)
	if got[idA] || got[idB] {
		t.Errorf("expected neither distant entity to record a candidate, got %v", got)
	}
}

func TestBroadPhase_Update_SingleEntity_SelfExcluded(t *testing.T) {
	space := testSpace(t)
	ecs := goke.New()
	var id uid.UID64
	var candidateQ *goke.Query
	var candidates goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		id = seedBroadPhaseEntity(t, si, space, posAt(0, 0, 10, 10))
		space.Flush(nil)
		candidateQ = si.NewQueryBuilder(&candidates).Build()
	}})

	bp := collisions.NewBroadPhase(space, testProbeMargin)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	if got := recorded(candidateQ, candidates); got[id] {
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

	bp := collisions.NewBroadPhase(space, testProbeMargin)
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
	var candidateQ *goke.Query
	var candidates goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedMovingCollidableEntity(t, si, space, posAt(0, 0, 10, 10), world.Velocity{Dir: geom.NewVec(1, 0), Value: 105})
		idB = seedBroadPhaseEntity(t, si, space, posAt(100, 0, 10, 10)) // far from idA's start, not yet overlapping
		space.Flush(nil)
		candidateQ = si.NewQueryBuilder(&candidates).Build()
	}})

	moveSystem := world.NewMoveSystem(space, 0)
	moveHandle := ecs.RegSys(moveSystem)
	bp := collisions.NewBroadPhase(space, testProbeMargin)
	bpHandle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(moveHandle, d)
		ctx.Run(bpHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second) // idA moves 105 units, landing 5px into idB

	got := recorded(candidateQ, candidates)
	if !got[idA] || !got[idB] {
		t.Errorf("expected BroadPhase to detect idA after MoveSystem moved it into idB, got %v (idA=%v idB=%v)", got, idA, idB)
	}
}

// The margin is not decoration: it is what lets the broad phase notice a pair
// one tick before they overlap. An entity travelling at world's own per-tick
// cap has to be recorded as touching on the tick *before* it arrives, or the
// narrow phase sees the overlap only once it is already deep.
//
// This is what the old fixed margin of 32 was guessing at. It guessed high,
// which cost candidates; guessing low would cost collisions.
func TestBroadPhase_MarginCoversOneTickOfClosingSpeed(t *testing.T) {
	const size = 10
	const maxStep = size / 2 // world's cap: MinSize/2
	margin := 2 * float64(maxStep)

	// A and B are exactly one tick of closing apart: still a clear gap now,
	// certain to overlap after both take their largest allowed step.
	gap := margin

	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var collQ *goke.Query
	var collTag goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, size, size))
		idB = seedBroadPhaseEntity(t, si, space, posAt(size+gap, 0, size, size))
		space.Flush(nil)
		collQ = si.NewQueryBuilder(&collTag).Build()
	}})

	bp := collisions.NewBroadPhase(space, margin)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	if !touches(collQ, &collTag, idA, idB) {
		t.Errorf("A did not record B as touching across a gap of %v — a pair this close closes it within one tick", gap)
	}
}

// The other half of the same contract: reach further than one tick of closing
// and every extra unit is area the index scans for pairs that cannot meet.
func TestBroadPhase_MarginStopsAtOneTickOfClosingSpeed(t *testing.T) {
	const size = 10
	const maxStep = size / 2
	margin := 2 * float64(maxStep)

	space := testSpace(t)
	ecs := goke.New()
	var idA, idB uid.UID64
	var collQ *goke.Query
	var collTag goke.Comp[collisions.Collision]

	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		idA = seedBroadPhaseEntity(t, si, space, posAt(0, 0, size, size))
		idB = seedBroadPhaseEntity(t, si, space, posAt(size+margin+1, 0, size, size))
		space.Flush(nil)
		collQ = si.NewQueryBuilder(&collTag).Build()
	}})

	bp := collisions.NewBroadPhase(space, margin)
	handle := ecs.RegSys(bp)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)

	if touches(collQ, &collTag, idA, idB) {
		t.Errorf("A recorded B as touching across a gap of %v — further than the pair can close in one tick", margin+1)
	}
}

// touches reports whether a's Collision lists b as a neighbour this tick.
func touches(q *goke.Query, tag *goke.Comp[collisions.Collision], a, b uid.UID64) bool {
	q.All()
	for q.Next() {
		cur := q.Cursor()
		tags := tag.Slice(cur)
		for i, id := range cur.IDs {
			if id != a {
				continue
			}
			for ti := uint8(0); ti < tags[i].TouchingCount; ti++ {
				if tags[i].Touching[ti] == b {
					return true
				}
			}
		}
	}
	return false
}
