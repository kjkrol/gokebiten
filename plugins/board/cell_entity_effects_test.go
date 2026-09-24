package board_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/effects"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/uid"
)

// Built WithEffects, the board drops a cell entity itself once its last effect ends, and the
// terrain keeps what the entity's Ground last said.
func TestCellEntity_GoesWhenItsLastEffectEndsWithEffects(t *testing.T) {
	grid := board.DefaultGrids{}.Square(4, 4, cellSize)
	target, _ := grid.CellIndex(2, 2)
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 4 * cellSize, Height: 4 * cellSize},
		Entities: world.EntitiesCfg{MaxCount: 4, MinSize: unitSize, MaxSize: unitSize},
	})
	fx := effects.NewPlugin(w)
	brd := board.NewPlugin(grid, &board.MultipleOccupancy{}, w).WithEffects(fx)
	grass := board.CellKind{Name: "grass", Cost: 1, Allows: board.Land}
	snow := board.CellKind{Name: "snow", Cost: 3, Allows: board.Land}
	brd.Res.Logic.Board.SetAll(grass)
	const tickLen = time.Second / 10
	frost := fx.Define("frost", effects.Spec{effects.Lasts(2 * tickLen), effects.Alter(func(g *board.Ground) { g.Kind = snow })})

	ctx := &installCtx{ecs: goke.New()}
	for _, install := range []func() error{
		func() error { return w.Install(ctx) }, func() error { return fx.Install(ctx) }, func() error { return brd.Install(ctx) },
	} {
		if err := install(); err != nil {
			t.Fatal(err)
		}
	}
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)
	var first uid.UID64
	cast := false
	caster := ctx.ecs.RegSys(goke.SystemFn{OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
		if !cast {
			first = brd.CellEntity(target)
			fx.Cast(cb, first, frost)
			cast = true
		}
	}})
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		rc.Run(caster, d)
		rc.Sync()
		w.RunPlan(rc, d)
		fx.RunPlan(rc, d)
		brd.RunPlan(rc, d)
		rc.Sync()
	})

	ctx.ecs.Tick(tickLen)
	if got := brd.Res.Logic.Board.Kind(target); got != snow {
		t.Fatalf("terrain is %q with the effect on, want snow", got.Name)
	}
	for range 4 {
		ctx.ecs.Tick(tickLen)
	}
	if got := brd.Res.Logic.Board.Kind(target); got != grass {
		t.Errorf("terrain is %q after the effect, want grass back", got.Name)
	}
	if again := brd.CellEntity(target); again == first {
		t.Error("the cell entity outlived its last effect; the board should have let it go")
	}
}

func TestCellEntity_OneEntityPerCellHoweverOftenAsked(t *testing.T) {
	grid := board.DefaultGrids{}.Square(4, 4, cellSize)
	a, _ := grid.CellIndex(1, 1)
	b, _ := grid.CellIndex(2, 2)
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 4 * cellSize, Height: 4 * cellSize},
		Entities: world.EntitiesCfg{MaxCount: 4, MinSize: unitSize, MaxSize: unitSize},
	})
	brd := board.NewPlugin(grid, &board.MultipleOccupancy{}, w)
	brd.Res.Logic.Board.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})

	ctx := &installCtx{ecs: goke.New()}
	for _, install := range []func() error{func() error { return w.Install(ctx) }, func() error { return brd.Install(ctx) }} {
		if err := install(); err != nil {
			t.Fatal(err)
		}
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

	first := brd.CellEntity(a)
	if again := brd.CellEntity(a); again != first {
		t.Errorf("cell a asked twice in one tick gave %d then %d", first, again)
	}
	other := brd.CellEntity(b)
	if other == first {
		t.Error("cells a and b share an entity")
	}
	ctx.ecs.Tick(time.Second / 10)
	if again := brd.CellEntity(a); again != first {
		t.Errorf("cell a after a tick gave %d, want %d", again, first)
	}
	if again := brd.CellEntity(b); again != other {
		t.Errorf("cell b after a tick gave %d, want %d", again, other)
	}
}
