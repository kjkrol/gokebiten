package collisions_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/uid"
)

type spyHandler struct {
	calls []collisions.CollisionEvent
}

func (s *spyHandler) OnCollision(_ *goke.CmdBuf, e collisions.CollisionEvent) {
	s.calls = append(s.calls, e)
}

func runNarrowPhase(t *testing.T, space *gokg.Space, handler collisions.CollisionHandler, setup ...goke.System) *goke.ECS {
	t.Helper()
	ecs := goke.New()
	ecs.Setup(setup...)

	np := collisions.NewNarrowPhase(space, handler, time.Hour)
	handle := ecs.RegSys(np)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(handle, d)
		ctx.Sync()
	})
	ecs.Tick(time.Millisecond)
	return ecs
}

// findPos scans q for id specifically and returns its Position — q may also
// match other entities, so this never trusts "the first thing the query
// finds" the way a bare range over q.Next() would.
func findPos(t *testing.T, q *goke.Query, posComp goke.Comp[kinematics.Position], id uid.UID64) kinematics.Position {
	t.Helper()
	q.All()
	for q.Next() {
		cur := q.Cursor()
		slice := posComp.Slice(cur)
		for i, gotID := range cur.IDs {
			if gotID == id {
				return slice[i]
			}
		}
	}
	t.Fatalf("entity %v not found by the query", id)
	return kinematics.Position{}
}

func TestNarrowPhase_DynamicDynamic_Overlap(t *testing.T) {
	space := testSpace(t)
	var idA, idB uid.UID64
	var posComp goke.Comp[kinematics.Position]
	var q *goke.Query
	handler := &spyHandler{}

	runNarrowPhase(t, space, handler, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var posA goke.Comp[kinematics.Position]
		var velA goke.Comp[kinematics.Velocity]
		var hitA goke.Comp[collisions.Hit]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &hitA, &collA)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10) // plenty of margin from the world edge
		idA = fa.IDs[0]

		var posB goke.Comp[kinematics.Position]
		var velB goke.Comp[kinematics.Velocity]
		fb := si.NewFactory(&posB, &velB)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10) // overlaps A by 5px on X
		idB = fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		posComp = posA
		q = si.NewQueryBuilder(&posA).Include(goke.Include[collisions.Collision]()).Build()
	}})

	if len(handler.calls) != 1 {
		t.Fatalf("handler called %d times, want 1", len(handler.calls))
	}
	e := handler.calls[0]
	if e.EntityA != idA || e.EntityB != idB {
		t.Errorf("event entities = (%v,%v), want (%v,%v)", e.EntityA, e.EntityB, idA, idB)
	}
	if e.Penetration.X == 0 && e.Penetration.Y == 0 {
		t.Errorf("expected a non-zero penetration vector, got %+v", e.Penetration)
	}

	if p := findPos(t, q, posComp, idA); p.TopLeft.X == 100 {
		t.Error("expected A's position to have moved apart from B")
	}
}

func TestNarrowPhase_DynamicStatic_OnlyDynamicMoves(t *testing.T) {
	space := testSpace(t)
	var posA, posB goke.Comp[kinematics.Position]
	var idA, idB uid.UID64
	var qA, qB *goke.Query
	var staticEditor *goke.Editor
	handler := &spyHandler{}

	seed := goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var velA goke.Comp[kinematics.Velocity]
		var hitA goke.Comp[collisions.Hit]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &hitA, &collA)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10) // plenty of margin from the world edge
		idA = fa.IDs[0]

		fb := si.NewFactory(&posB) // no Velocity -> must be tagged Static (see addStatic below)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10)
		idB = fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		qA = si.NewQueryBuilder(&posA).Include(goke.Include[kinematics.Velocity]()).Build()
		qB = si.NewQueryBuilder(&posB).Exclude(goke.Exclude[kinematics.Velocity]()).Build()
	}}

	// Static is a zero-size tag: Factory/Track reject zero-size data
	// columns, so — same as Sensor — it's added via an Editor migration in
	// a second Setup-phase system run after the seed.
	var staticComp goke.Comp[collisions.Static]
	addStatic := goke.SystemFn{
		OnInit: func(si *goke.SysInit) {
			staticEditor = qB.NewEditorBuilder(&staticComp).Build()
		},
		OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
			qB.All()
			for qB.Next() {
				buf := qB.BeginMigrate(cb)
				for _, id := range qB.Cursor().IDs {
					buf.Add(id)
				}
				buf.Commit(staticEditor)
			}
		},
	}

	runNarrowPhase(t, space, handler, seed, addStatic)

	if len(handler.calls) != 1 {
		t.Fatalf("handler called %d times, want 1", len(handler.calls))
	}
	if p := findPos(t, qA, posA, idA); p.TopLeft.X == 100 {
		t.Error("expected dynamic A to have moved")
	}
	if p := findPos(t, qB, posB, idB); p.TopLeft.X != 105 {
		t.Errorf("expected static B to stay put at X=105, got X=%d", p.TopLeft.X)
	}
}

func TestNarrowPhase_SensorContact_SkipsHandlerAndPush(t *testing.T) {
	space := testSpace(t)
	var posA goke.Comp[kinematics.Position]
	var idA uid.UID64
	var qA *goke.Query
	var sensorEditor *goke.Editor
	handler := &spyHandler{}

	seed := goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var velA goke.Comp[kinematics.Velocity]
		var hitA goke.Comp[collisions.Hit]
		var collA goke.Comp[collisions.Collision]
		fa := si.NewFactory(&posA, &velA, &hitA, &collA)
		fa.Create(1)
		fa.Next()
		posA.Slice(&fa.Cursor)[0] = posAt(100, 100, 10, 10) // plenty of margin from the world edge
		idA = fa.IDs[0]

		var posB goke.Comp[kinematics.Position]
		var velB goke.Comp[kinematics.Velocity]
		fb := si.NewFactory(&posB, &velB)
		fb.Create(1)
		fb.Next()
		posB.Slice(&fb.Cursor)[0] = posAt(105, 100, 10, 10)
		idB := fb.IDs[0]

		coll := collA.Slice(&fa.Cursor)
		coll[0].Touching[0] = idB
		coll[0].TouchingCount = 1

		qA = si.NewQueryBuilder(&posA).Include(goke.Include[kinematics.Velocity]()).Build()
	}}

	// Sensor is a zero-size tag: Factory/Track reject zero-size data columns,
	// so it has to be added via an Editor migration (comp.Add has no such
	// restriction), in a second Setup-phase system run after the seed.
	var sensorComp goke.Comp[collisions.Sensor]
	addSensor := goke.SystemFn{
		OnInit: func(si *goke.SysInit) {
			sensorEditor = qA.NewEditorBuilder(&sensorComp).Build()
		},
		OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
			qA.All()
			for qA.Next() {
				buf := qA.BeginMigrate(cb)
				for _, id := range qA.Cursor().IDs {
					buf.Add(id)
				}
				buf.Commit(sensorEditor)
			}
		},
	}

	runNarrowPhase(t, space, handler, seed, addSensor)

	if len(handler.calls) != 0 {
		t.Errorf("handler called %d times, want 0 (sensor contacts must not dispatch to the handler)", len(handler.calls))
	}
	if p := findPos(t, qA, posA, idA); p.TopLeft.X != 100 {
		t.Errorf("expected a sensor contact to never physically push A, TopLeft.X = %d, want 100", p.TopLeft.X)
	}
}

func TestNarrowPhase_NoTouching_RemovesUnconfirmedHitTag(t *testing.T) {
	space := testSpace(t)
	var hitTag goke.Comp[collisions.Hit]
	var q *goke.Query

	runNarrowPhase(t, space, &spyHandler{}, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[kinematics.Position]
		var vel goke.Comp[kinematics.Velocity]
		var coll goke.Comp[collisions.Collision]
		f := si.NewFactory(&pos, &vel, &hitTag, &coll)
		f.Create(1)
		f.Next()
		pos.Slice(&f.Cursor)[0] = posAt(0, 0, 10, 10)
		// TouchingCount left at 0 -> never paired, never confirmed this tick.

		q = si.NewQueryBuilder(&hitTag).Build()
	}})

	q.All()
	found := false
	for q.Next() {
		found = found || len(q.Cursor().IDs) > 0
	}
	if found {
		t.Error("expected the never-confirmed Hit tag to be removed")
	}
}

func TestNarrowPhase_AlreadyExpiring_SurvivesUnconfirmedTick(t *testing.T) {
	space := testSpace(t)
	var hitTag goke.Comp[collisions.Hit]
	var q *goke.Query

	runNarrowPhase(t, space, &spyHandler{}, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[kinematics.Position]
		var vel goke.Comp[kinematics.Velocity]
		var coll goke.Comp[collisions.Collision]
		f := si.NewFactory(&pos, &vel, &hitTag, &coll)
		f.Create(1)
		f.Next()
		pos.Slice(&f.Cursor)[0] = posAt(0, 0, 10, 10)
		hitTag.Slice(&f.Cursor)[0].SetExpiresAt(time.Now().Add(time.Hour))
		// TouchingCount left at 0 -> unconfirmed this tick, but already expiring.

		q = si.NewQueryBuilder(&hitTag).Build()
	}})

	q.All()
	found := false
	for q.Next() {
		found = found || len(q.Cursor().IDs) > 0
	}
	if !found {
		t.Error("expected an already-expiring Hit tag to survive one unconfirmed tick (grace period until it actually expires)")
	}
}
