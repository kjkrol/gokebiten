package world

import (
	"testing"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

func TestSpawn_AnEntitySeededOnTheSeamIsWholeBeforeTheFirstTick(t *testing.T) {
	wm := NewPlugin(Config{
		Space:    SpaceCfg{Width: 1000, Height: 1000, Edges: aabbworld.Torus},
		Entities: EntitiesCfg{MaxCount: 10, MinSize: 1, MaxSize: 100},
	}).module
	wm.populate(testKind(
		Position{AABB: plane.NewAABB(geom.NewVec(990, 500), 20, 20)},
		Velocity{},
	), []any{nil})

	var base goke.Comp[Base]
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&base).Build()
	}})...)

	across := geom.NewAABB(geom.NewVec(2, 505), geom.NewVec(8, 510))
	if n := wm.space.Query(across, aabbworld.AnyCapability, func(uid.UID64) {}); n != 1 {
		t.Errorf("Query finds %d pieces across the seam before any tick, want the entity", n)
	}
	for query.All(); query.Next(); {
		for _, b := range base.Slice(query.Cursor()) {
			if b.Pos.Overhang.X != 10 {
				t.Errorf("Base.Pos overhang %v, want 10 past the right edge", b.Pos.Overhang)
			}
		}
	}
}
