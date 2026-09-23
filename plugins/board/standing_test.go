package board_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

const tickLen = time.Second / 60

// installWorldAndBoard installs w and brd alone, with one land unit at cell (1,1), and returns
// the ECS ticking world then board.
func installWorldAndBoard(t *testing.T, w *world.Plugin, brd *board.Plugin, grid board.Grid) *goke.ECS {
	t.Helper()
	ctx := &installCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := brd.Install(ctx); err != nil {
		t.Fatal(err)
	}
	start, _ := grid.CellIndex(1, 1)
	w.Seed(kind.Define[mover](w.Kinds(), "unit", kind.Spec{
		kind.Load(func(m mover) world.Position { return world.Position{AABB: board.CellAABB(grid, m.cell, unitSize)} }),
		kind.Const(world.Velocity{}),
		kind.Load(func(m mover) board.Cell { return board.Cell{ID: m.cell} }),
		kind.Const(board.Mover{Domain: board.Land}),
	}).Entry(mover{cell: start}))
	if err := w.Populate(); err != nil {
		t.Fatal(err)
	}
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		w.RunPlan(rc, d)
		brd.RunPlan(rc, d)
		rc.Sync()
	})
	return ctx.ecs
}

// footing records the last Standing of every entity a behavior saw, and whether it fell.
type footing struct {
	last map[uid.UID64]board.Standing
	fell map[uid.UID64]bool
}

func newFooting() *footing {
	return &footing{last: map[uid.UID64]board.Standing{}, fell: map[uid.UID64]bool{}}
}

func (f *footing) react(_ plugin.Tick, m *board.Mover, st board.Standing) {
	f.last[st.ID] = st
	f.fell[st.ID] = st.Fell(m.Domain)
}

// pitBoard is grass with a pit of kind pit down column 3.
func pitBoard(grid board.Grid, pit board.CellKind) func(*board.Board) {
	return func(brd *board.Board) {
		brd.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
		for y := uint32(1); y <= 14; y++ {
			c, _ := grid.CellIndex(3, y)
			brd.Set(c, pit)
		}
	}
}

func TestStanding_ALandUnitDrivenIntoAHoleFellAndKeepsFalling(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	start, _ := grid.CellIndex(1, 7)
	f := newFooting()
	bw := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, pitBoard(grid, board.CellKind{Name: "hole", Cost: 1}),
		[]mover{{cell: start, heading: east}}, plugin.Each[board.Mover](f.react))
	if bodies, _ := bw.snapshot(); len(bodies) != 0 {
		t.Fatalf("%d bodies, want none: a hole is not solid", len(bodies))
	}

	fellAt := -1
	for tick := range 120 {
		bw.tick()
		_, units := bw.snapshot()
		centre := board.Center(world.Position{AABB: toPlane(units[0])})
		over, _ := grid.CellAt(centre)
		id := onlyID(f)
		switch {
		case bw.brd.Res.Logic.Board.Kind(over).Name == "hole" && !f.fell[id]:
			t.Fatalf("tick %d: centre over the hole at %v, but the unit did not fall", tick, over)
		case bw.brd.Res.Logic.Board.Kind(over).Name != "hole" && f.fell[id]:
			t.Fatalf("tick %d: centre over %s, but the unit fell", tick, f.last[id].Kind.Name)
		case f.fell[id] && fellAt < 0:
			fellAt = tick
		}
		if f.fell[id] && f.last[id].Cell != over {
			t.Fatalf("tick %d: Standing names cell %v, the centre is over %v", tick, f.last[id].Cell, over)
		}
	}
	if fellAt < 0 {
		t.Fatal("the unit never reached the hole")
	}
}

func TestStanding_ABoatOnWaterHasNotFallen(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	start, _ := grid.CellIndex(3, 7)
	f := newFooting()
	bw := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, pitBoard(grid, board.CellKind{Name: "water", Cost: 1, Allows: board.Water}),
		[]mover{{cell: start, domain: board.Water}}, plugin.Each[board.Mover](f.react))
	bw.tick()
	id := onlyID(f)
	if f.last[id].Kind.Name != "water" || f.fell[id] {
		t.Errorf("a boat on %s fell=%v, want water and no fall", f.last[id].Kind.Name, f.fell[id])
	}
}

func TestStanding_ReportsEveryUnitOnTheBoardAndNoBody(t *testing.T) {
	f := newFooting()
	bw, _ := squareWorldWith(t, plugin.Each[board.Mover](f.react), mover{})
	bw.tick()
	bodies, units := bw.snapshot()
	if len(f.last) != len(units) || len(bodies) == 0 {
		t.Fatalf("%d standings for %d units beside %d bodies", len(f.last), len(units), len(bodies))
	}
	for id, st := range f.last {
		if bw.isBody(id) {
			t.Errorf("body %d got a Standing", id)
		}
		if st.Kind.Name != "grass" || f.fell[id] {
			t.Errorf("unit %d stands on %s, fell=%v; want grass, no fall", id, st.Kind.Name, f.fell[id])
		}
	}
}

func TestStanding_WorksWithoutCollision(t *testing.T) {
	grid := board.DefaultGrids{}.Square(4, 4, cellSize)
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 4 * cellSize, Height: 4 * cellSize},
		Entities: world.EntitiesCfg{MaxCount: 2, MinSize: unitSize, MaxSize: unitSize},
	})
	brd := board.NewPlugin(grid, &board.MultipleOccupancy{}, w)
	brd.Res.Logic.Board.SetAll(board.CellKind{Name: "hole", Cost: 1})
	f := newFooting()
	if err := brd.RegisterBehavior(plugin.Each[board.Mover](f.react)); err != nil {
		t.Fatal(err)
	}
	if err := brd.RegisterBehavior(plugin.Between[board.Mover, board.Mover](func(plugin.Tick, board.Standing) {})); !errors.Is(err, plugin.ErrUnhostedBehavior) {
		t.Errorf("Between on board: %v, want ErrUnhostedBehavior", err)
	}
	if err := brd.RegisterBehavior(plugin.Each[board.Mover](func(plugin.Tick, *board.Mover, collision.Struck) {})); !errors.Is(err, plugin.ErrUnhostedBehavior) {
		t.Errorf("Each of Struck on board: %v, want ErrUnhostedBehavior", err)
	}
	ecs := installWorldAndBoard(t, w, brd, grid)
	ecs.Tick(tickLen)
	if len(f.fell) != 1 {
		t.Fatalf("%d standings, want 1", len(f.fell))
	}
	for _, fell := range f.fell {
		if !fell {
			t.Error("a land unit spawned over a hole did not fall")
		}
	}
	if err := brd.RegisterBehavior(plugin.Each[board.Mover](f.react)); !errors.Is(err, plugin.ErrHostBuilt) {
		t.Errorf("registering after Setup: %v, want ErrHostBuilt", err)
	}
}

// onlyID is the one entity the footing saw.
func onlyID(f *footing) uid.UID64 {
	for id := range f.last {
		return id
	}
	return 0
}

func toPlane(b geom.AABB) plane.AABB {
	size := b.BottomRight.Sub(b.TopLeft)
	return plane.NewAABB(b.TopLeft, size.X, size.Y)
}
