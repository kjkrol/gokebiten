package board_test

import (
	"testing"

	"github.com/kjkrol/goke/v3"

	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/world"
)

func TestTerrainSpeedModifier_ScalesByOneOverCost(t *testing.T) {
	grid := board.DefaultGrids{}.Square(3, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Allows: board.Land})
	at := func(x uint32) board.CellID { c, _ := grid.CellIndex(x, 0); return c }
	terrain.Set(at(0), board.CellKind{Cost: 2, Allows: board.Land})   // slow
	terrain.Set(at(1), board.CellKind{Cost: 0.5, Allows: board.Land}) // a boost, if a game wants one
	terrain.Set(at(2), board.CellKind{Cost: 0, Allows: board.Land})   // no cost: no effect
	m := board.NewTerrainSpeedModifier(grid, terrain)

	// Entities without a Mover: the modifier charges them as Land.
	var base goke.Comp[world.Base]
	got := map[float64]float64{} // by x position
	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		qb := si.NewQueryBuilder(&base)
		m.Bind(qb)
		q := qb.Build()
		f := si.NewFactory(&base)
		f.Create(3)
		x := uint32(0)
		for f.Next() {
			for i := range f.Cursor.IDs {
				base.Slice(&f.Cursor)[i] = world.Base{Pos: world.Position{AABB: board.CellAABB(grid, at(x), 4)}}
				x++
			}
		}
		for q.All(); q.Next(); {
			cur := q.Cursor()
			bases := base.Slice(cur)
			for i := range cur.IDs {
				got[bases[i].Pos.TopLeft.X] = m.Apply(cur, i, &bases[i], 1)
			}
		}
	}})
	for x, want := range map[uint32]float64{0: 0.5, 1: 2, 2: 1} {
		x0 := board.CellAABB(grid, at(x), 4).TopLeft.X
		if got[x0] != want {
			t.Errorf("cell %d: factor %v, want %v", x, got[x0], want)
		}
	}
}

func TestTerrainSpeedModifier_ChargesTheEntitysOwnDomain(t *testing.T) {
	const frost = board.Domain(1 << 3)
	grid := board.DefaultGrids{}.Square(1, 1, 10)
	terrain := board.NewTerrainMap()
	c, _ := grid.CellIndex(0, 0)
	terrain.Set(c, board.CellKind{Name: "snow", Cost: 4, Allows: board.Land | frost}.Costing(frost, 0.5))
	m := board.NewTerrainSpeedModifier(grid, terrain)

	// Two entities on the snow, told apart by a one-unit offset: on foot, and frost-born.
	var mover goke.Comp[board.Mover]
	var base goke.Comp[world.Base]
	ecs := goke.New()
	got := map[float64]float64{} // by x offset
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		qb := si.NewQueryBuilder(&base)
		m.Bind(qb)
		q := qb.Build()
		f := si.NewFactory(&base, &mover)
		f.Create(2)
		domains := []board.Domain{board.Land, board.Land | frost}
		for f.Next() {
			for i := range f.Cursor.IDs {
				box := board.CellAABB(grid, c, 4)
				box.TopLeft.X += float64(2 - len(domains))
				base.Slice(&f.Cursor)[i] = world.Base{Pos: world.Position{AABB: box}}
				mover.Slice(&f.Cursor)[i] = board.Mover{Domain: domains[0]}
				domains = domains[1:]
			}
		}
		for q.All(); q.Next(); {
			cur := q.Cursor()
			bases := base.Slice(cur)
			for i := range cur.IDs {
				got[bases[i].Pos.TopLeft.X] = m.Apply(cur, i, &bases[i], 1)
			}
		}
	}})
	x0 := board.CellAABB(grid, c, 4).TopLeft.X
	if got[x0] != 0.25 || got[x0+1] != 2 {
		t.Errorf("factors land=%v frost=%v, want 0.25 and 2", got[x0], got[x0+1])
	}
}
