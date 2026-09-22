package world_test

import (
	"testing"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/world"
)

func TestPosition_RoundTrip(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	ecs := goke.New()
	var base goke.Comp[world.Base]
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&base)
		f.Create(1)
		f.Next()
		p := base.Slice(&f.Cursor)
		p[0].Pos = world.Position{AABB: plane.NewAABB(geom.NewVec(12, 34), 5, 6)}
	}})

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ecs2 := goke.New()
	if err := ecs2.Load(path, goke.LoadComp[world.Base]()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	var base2 goke.Comp[world.Base]
	var q *goke.Query
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&base2).Build()
	}})
	q.All()
	found := false
	for q.Next() {
		p := base2.Slice(q.Cursor())
		for i := range p {
			found = true
			if p[i].Pos.TopLeft.X != 12 || p[i].Pos.TopLeft.Y != 34 {
				t.Errorf("TopLeft = %+v, want (12,34)", p[i].Pos.TopLeft)
			}
			if p[i].Pos.Size.X != 5 || p[i].Pos.Size.Y != 6 {
				t.Errorf("Size = %+v, want (5,6)", p[i].Pos.Size)
			}
		}
	}
	if !found {
		t.Fatal("no entity found after Load")
	}
}
