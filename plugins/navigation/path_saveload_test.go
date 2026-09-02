package navigation_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/navigation"
)

// TestMoveTo_RoundTrip guards navigation.MoveOrder's nested Path (a fixed-size
// array field) against goke's slice rejection — see
// internal/comp.ValidateEncodable — confirming it actually survives
// Save/Load rather than just satisfying RegComp.
func TestMoveTo_RoundTrip(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	want := navigation.MoveOrder{Target: board.CellID(7), Path: navigation.Path{Length: 3, Index: 1}}
	want.Path.Steps[0] = board.CellID(10)
	want.Path.Steps[1] = board.CellID(11)
	want.Path.Steps[2] = board.CellID(12)

	ecs := goke.New()
	var moveToComp goke.Comp[navigation.MoveOrder]
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&moveToComp)
		f.Create(1)
		f.Next()
		moveToComp.Slice(&f.Cursor)[0] = want
	}})

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ecs2 := goke.New()
	if err := ecs2.Load(path, goke.LoadComp[navigation.MoveOrder]()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	var moveToComp2 goke.Comp[navigation.MoveOrder]
	var q *goke.Query
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&moveToComp2).Build()
	}})
	q.All()
	found := false
	for q.Next() {
		got := moveToComp2.Slice(q.Cursor())
		for i := range got {
			found = true
			if got[i] != want {
				t.Errorf("MoveOrder = %+v, want %+v", got[i], want)
			}
		}
	}
	if !found {
		t.Fatal("no entity found after Load")
	}
}
