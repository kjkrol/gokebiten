package board_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
)

// TestPath_RoundTrip guards board.Path's fixed-size-array shape against
// goke's slice rejection — see internal/comp.ValidateEncodable — confirming
// it actually survives Save/Load rather than just satisfying RegComp.
func TestPath_RoundTrip(t *testing.T) {
	path := t.TempDir() + "/save.bin"

	want := board.Path{Length: 3, Index: 1}
	want.Steps[0] = board.CellID(10)
	want.Steps[1] = board.CellID(11)
	want.Steps[2] = board.CellID(12)

	ecs := goke.New()
	var pathComp goke.Comp[board.Path]
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&pathComp)
		f.Create(1)
		f.Next()
		pathComp.Slice(&f.Cursor)[0] = want
	}})

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ecs2 := goke.New()
	if err := ecs2.Load(path, goke.LoadComp[board.Path]()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	var pathComp2 goke.Comp[board.Path]
	var q *goke.Query
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&pathComp2).Build()
	}})
	q.All()
	found := false
	for q.Next() {
		got := pathComp2.Slice(q.Cursor())
		for i := range got {
			found = true
			if got[i] != want {
				t.Errorf("Path = %+v, want %+v", got[i], want)
			}
		}
	}
	if !found {
		t.Fatal("no entity found after Load")
	}
}
