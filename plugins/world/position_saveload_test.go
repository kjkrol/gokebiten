package world_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// TestPosition_RoundTrip guards against a real bug that shipped briefly:
// plane.AABB (embedded, anonymous, in world.Position) once implemented
// BinaryMarshaler for its own reasons — Go's method promotion made world.Position
// implement it too, so goke's persist treated all of world.Position as one opaque
// blob written by the promoted method, which only knows about AABB's own
// fields, silently dropping the rest on every Save. Fixed by keeping AABB a
// plain, recursively POD-encodable type — goke also now rejects this
// pattern generally at RegComp (internal/comp.ValidateEncodable's
// promotedCodecHazard check).
func TestPosition_RoundTrip(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	ecs := goke.New()
	var pos goke.Comp[world.Position]
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&pos)
		f.Create(1)
		f.Next()
		p := pos.Slice(&f.Cursor)
		p[0] = world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](12, 34), 5, 6)}
	}})

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ecs2 := goke.New()
	if err := ecs2.Load(path, goke.LoadComp[world.Position]()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	var pos2 goke.Comp[world.Position]
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
			if p[i].TopLeft.X != 12 || p[i].TopLeft.Y != 34 {
				t.Errorf("TopLeft = %+v, want (12,34)", p[i].TopLeft)
			}
			if p[i].Size.X != 5 || p[i].Size.Y != 6 {
				t.Errorf("Size = %+v, want (5,6)", p[i].Size)
			}
		}
	}
	if !found {
		t.Fatal("no entity found after Load")
	}
}
