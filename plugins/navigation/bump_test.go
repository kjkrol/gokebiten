package navigation

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/selection"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

// A road one cell wide through a field, SingleOccupancy and collision as island-demo has them: the
// scene of the reported deadlock, where units pushed each other for ever.
type roadUnit struct {
	start, target board.CellID
	ordered       bool
	domain        board.Domain // zero: Land
}

type roadWorld struct {
	t     *testing.T
	grid  board.Grid
	ecs   *goke.ECS
	nav   *Plugin
	cell  goke.Comp[board.Cell]
	base  goke.Comp[world.Base]
	order goke.OptComp[MoveOrder]
	q     *goke.Query
	kinds map[uid.UID64]kind.ID
	byRow map[int]uid.UID64 // row index → entity, in the order given
}

const roadCell = 32

func newRoadWorld(t *testing.T, width uint32, units []roadUnit) *roadWorld {
	t.Helper()
	rw := &roadWorld{t: t, grid: board.DefaultGrids{}.Square(width, 3, roadCell), byRow: map[int]uid.UID64{}}
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: width * roadCell, Height: 3 * roadCell},
		Entities: world.EntitiesCfg{MaxCount: len(units), MinSize: 22, MaxSize: 22},
	})
	occupancy := &board.SingleOccupancy{}
	c := collision.NewPlugin(w)
	brd := board.NewPlugin(rw.grid, occupancy, w).WithCollision(c)
	brd.Res.Logic.Board.SetAll(board.CellKind{Cost: 2, Allows: board.Land | board.Air}) // field
	for x := uint32(0); x < width; x++ {
		brd.Res.Logic.Board.Set(rw.at(x, 1), board.CellKind{Cost: 1, Allows: board.Land | board.Air}) // the road
	}
	sel := selection.NewPlugin(w)
	rw.nav = NewPlugin(brd, w, sel)

	ctx := &stubInstallCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := brd.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := rw.nav.Install(ctx); err != nil {
		t.Fatal(err)
	}

	spec := func(ordered bool, domain board.Domain) kind.Spec {
		if domain == 0 {
			domain = board.Land
		}
		s := kind.Spec{
			kind.Load(func(u roadUnit) world.Position { return world.Position{AABB: board.CellAABB(rw.grid, u.start, 22)} }),
			kind.Const(world.Velocity{}),
			kind.Const(world.Steering{MaxSpeed: 96, Accel: 192, Brake: 384, V0: 48, TurnRate: 0.15}),
			kind.Load(func(u roadUnit) board.Cell { return board.Cell{ID: u.start} }),
			kind.Const(collision.Collider{Layers: uint8(domain)}),
			kind.Const(collision.Physics{}),
			kind.Const(board.Mover{Domain: domain}),
		}
		if ordered {
			s = append(s, kind.Load(func(u roadUnit) MoveOrder { return MoveOrder{Target: u.target} }))
		}
		return s
	}
	kindIDs := make([]kind.ID, len(units))
	for i, u := range units {
		k := kind.Define[roadUnit](w.Kinds(), string(rune('a'+i)), spec(u.ordered, u.domain))
		kindIDs[i] = k.ID()
		w.Seed(k.Entry(u))
	}
	if err := w.Populate(); err != nil {
		t.Fatal(err)
	}

	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		rw.q = si.NewQueryBuilder(&rw.cell, &rw.base).Optional(&rw.order).Build()
	}})
	ctx.ecs.Setup(systems...)
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		w.RunPlan(rc, d)
		c.RunPlan(rc, d)
		brd.RunPlan(rc, d)
		rw.nav.RunPlan(rc, d)
		rc.Sync()
	})
	rw.ecs = ctx.ecs
	for rw.q.All(); rw.q.Next(); {
		cur := rw.q.Cursor()
		for i, id := range cur.IDs {
			for row, k := range kindIDs {
				if rw.base.Slice(cur)[i].TypeID == k {
					rw.byRow[row] = id
				}
			}
		}
	}
	return rw
}

func (rw *roadWorld) at(x, y uint32) board.CellID { c, _ := rw.grid.CellIndex(x, y); return c }

// state is one unit's cell and order, if it still has one.
func (rw *roadWorld) state(id uid.UID64) (cell board.CellID, order *MoveOrder) {
	for rw.q.All(); rw.q.Next(); {
		cur := rw.q.Cursor()
		for i, got := range cur.IDs {
			if got != id {
				continue
			}
			cell = rw.cell.Slice(cur)[i].ID
			if orders := rw.order.Slice(cur); orders != nil {
				o := orders[i]
				order = &o
			}
		}
	}
	return
}

// run ticks until every ordered unit has arrived — on its target, or beside it when someone stands
// there — or the time is up; it reports the ticks taken and how many times each unit's route changed.
func (rw *roadWorld) run(units []roadUnit, limit time.Duration) (ticks int, replans map[uid.UID64]int) {
	replans = map[uid.UID64]int{}
	last := map[uid.UID64][]board.CellID{}
	for ticks = 0; time.Duration(ticks)*time.Second/60 < limit; ticks++ {
		rw.ecs.Tick(time.Second / 60)
		done := true
		for row, u := range units {
			if !u.ordered {
				continue
			}
			id := rw.byRow[row]
			cell, o := rw.state(id)
			if o == nil {
				if cell != u.target && rw.grid.Distance(cell, u.target) > 1.5 {
					rw.t.Fatalf("unit %d lost its order at %v, neither on nor beside its target %v", id, cell, u.target)
				}
				continue
			}
			done = false
			steps := append([]board.CellID(nil), o.Path.Steps[:o.Path.Length]...)
			if !equalSteps(steps, last[id]) {
				replans[id]++
				last[id] = steps
			}
		}
		if done {
			return ticks, replans
		}
	}
	return ticks, replans
}

func TestBump_TwoUnitsHeadOnOnARoadPassEachOther(t *testing.T) {
	units := []roadUnit{{start: 0, ordered: true}, {start: 0, ordered: true}}
	rw := newRoadWorld(t, 10, units)
	units[0] = roadUnit{start: rw.at(0, 1), target: rw.at(9, 1), ordered: true}
	units[1] = roadUnit{start: rw.at(9, 1), target: rw.at(0, 1), ordered: true}
	rw = newRoadWorld(t, 10, units)

	ticks, replans := rw.run(units, 10*time.Second)
	if ticks >= 60*10 {
		t.Fatalf("the two units did not both arrive within 10 s; replans %v", replans)
	}
	for id, n := range replans {
		if n > 10 {
			t.Errorf("unit %d re-planned %d times, want a handful: one per bump, at most one per bumpInterval", id, n)
		}
	}
}

func TestBump_ThreeInARowUntangle(t *testing.T) {
	rw := newRoadWorld(t, 12, []roadUnit{{start: 0, ordered: true}})
	units := []roadUnit{
		{start: rw.at(0, 1), target: rw.at(11, 1), ordered: true},
		{start: rw.at(1, 1), target: rw.at(11, 1), ordered: true},
		{start: rw.at(11, 1), target: rw.at(0, 1), ordered: true},
	}
	rw = newRoadWorld(t, 12, units)
	if ticks, replans := rw.run(units, 12*time.Second); ticks >= 60*12 {
		t.Fatalf("three units did not all arrive within 12 s; replans %v", replans)
	}
}

func TestBump_ATargetSomeoneStandsOnIsSettledBeside(t *testing.T) {
	rw := newRoadWorld(t, 6, []roadUnit{{start: 0, ordered: true}})
	units := []roadUnit{
		{start: rw.at(0, 1), target: rw.at(4, 1), ordered: true},
		{start: rw.at(4, 1)}, // standing on the target, going nowhere
	}
	rw = newRoadWorld(t, 6, units)
	for tick := range 60 * 4 {
		rw.ecs.Tick(time.Second / 60)
		cell, o := rw.state(rw.byRow[0])
		if o == nil {
			if cell == rw.at(4, 1) || rw.grid.Distance(cell, rw.at(4, 1)) > 1.5 {
				t.Fatalf("tick %d: settled at %v, want a cell beside the occupied target", tick, cell)
			}
			return
		}
	}
	t.Fatal("the unit never settled beside the occupied target within 4 s")
}

// A hawk over a road full of walkers flies straight through: they hold the Land layer of their
// cells, it moves in Air, and neither the planner, the leg nor the solver mind the other.
func TestBump_AFlyerPassesOverWalkersUntouched(t *testing.T) {
	rw := newRoadWorld(t, 8, []roadUnit{{start: 0, ordered: true}})
	units := []roadUnit{
		{start: rw.at(0, 1), target: rw.at(7, 1), ordered: true, domain: board.Air},
		{start: rw.at(3, 1)}, // walkers parked across the road
		{start: rw.at(4, 1)},
		{start: rw.at(5, 1)},
	}
	rw = newRoadWorld(t, 8, units)
	hawk := rw.byRow[0]
	_, o := rw.state(hawk)
	if o == nil {
		t.Fatal("the hawk has no order")
	}
	ticks, replans := rw.run(units[:1], 6*time.Second)
	if ticks >= 60*6 {
		t.Fatalf("the hawk did not reach the far end within 6 s; replans %v", replans)
	}
	if replans[hawk] > 1 {
		t.Errorf("the hawk re-planned %d times, want its first straight route kept: nothing on the Land layer stops it", replans[hawk])
	}
	for _, row := range []int{1, 2, 3} {
		if cell, _ := rw.state(rw.byRow[row]); cell != units[row].start {
			t.Errorf("walker %d was moved to %v by the hawk passing over", row, cell)
		}
	}
}
