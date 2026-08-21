package kinematics_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
)

// TestPosition_AccXY_RoundTrip guards against a real bug that shipped
// briefly: plane.AABB (embedded, anonymous, in Position) once implemented
// BinaryMarshaler for its own reasons — Go's method promotion made Position
// implement it too, so goke's persist treated all of Position as one opaque
// blob written by the promoted method, which only knows about AABB's own
// fields. AccX/AccY were silently dropped on every Save. Fixed by keeping
// AABB a plain, recursively POD-encodable type instead (see plane.AABB's
// Frags/FragMask) — goke also now rejects this pattern generally at
// RegComp (internal/comp.ValidateEncodable's promotedCodecHazard check).
func TestPosition_AccXY_RoundTrip(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	ecs := goke.New()
	var pos goke.Comp[kinematics.Position]
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&pos)
		f.Create(1)
		f.Next()
		p := pos.Slice(&f.Cursor)
		p[0].AccX = 123.456
		p[0].AccY = 789.012
	}})

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ecs2 := goke.New()
	if err := ecs2.Load(path, goke.LoadComp[kinematics.Position]()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	var pos2 goke.Comp[kinematics.Position]
	var q *goke.Query
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&pos2).Build()
	}})
	q.All()
	found := false
	for q.Next() {
		p := pos2.Slice(q.Cursor())
		for i := range p {
			found = true
			if p[i].AccX != 123.456 || p[i].AccY != 789.012 {
				t.Errorf("AccX/AccY = %v/%v, want 123.456/789.012", p[i].AccX, p[i].AccY)
			}
		}
	}
	if !found {
		t.Fatal("no entity found after Load")
	}
}
