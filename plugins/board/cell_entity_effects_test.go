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

// cellWorld is world + effects + board over a 4x4 grass board, with frost defined, casting from a
// hook at the start of each tick; boardFirst runs the board's pass before the effects'.
type cellWorld struct {
	ecs     *goke.ECS
	brd     *board.Plugin
	fx      *effects.Plugin
	frost   effects.ID
	target  board.CellID
	grass   board.CellKind
	snow    board.CellKind
	casting func(cb *goke.CmdBuf)
}

const cellTick = time.Second / 10

func newCellWorld(t *testing.T, boardFirst bool) *cellWorld {
	t.Helper()
	cw := &cellWorld{}
	grid := board.DefaultGrids{}.Square(4, 4, cellSize)
	cw.target, _ = grid.CellIndex(2, 2)
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 4 * cellSize, Height: 4 * cellSize},
		Entities: world.EntitiesCfg{MaxCount: 4, MinSize: unitSize, MaxSize: unitSize},
	})
	cw.fx = effects.NewPlugin(w)
	cw.brd = board.NewPlugin(grid, &board.MultipleOccupancy{}, w)
	cw.grass = board.CellKind{Name: board.Named("grass"), Cost: 1, Allows: board.Land}
	cw.snow = board.CellKind{Name: board.Named("snow"), Cost: 3, Allows: board.Land}
	cw.brd.Res.Logic.Board.SetAll(cw.grass)
	snow := cw.snow
	cw.frost = cw.fx.Define("frost", effects.Spec{effects.Lasts(2 * cellTick), effects.Alter(func(g *board.Ground) { g.Kind = snow })})

	ctx := &installCtx{ecs: goke.New()}
	for _, install := range []func() error{
		func() error { return w.Install(ctx) }, func() error { return cw.fx.Install(ctx) }, func() error { return cw.brd.Install(ctx) },
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
	caster := ctx.ecs.RegSys(goke.SystemFn{OnUpdate: func(cb *goke.CmdBuf, _ time.Duration) {
		if cw.casting != nil {
			cw.casting(cb)
			cw.casting = nil
		}
	}})
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		rc.Run(caster, d)
		rc.Sync()
		w.RunPlan(rc, d)
		if boardFirst {
			cw.brd.RunPlan(rc, d)
			cw.fx.RunPlan(rc, d)
		} else {
			cw.fx.RunPlan(rc, d)
			cw.brd.RunPlan(rc, d)
		}
		rc.Sync()
	})
	cw.ecs = ctx.ecs
	return cw
}

// castFrost queues a frost on the target's cell entity for the next tick and returns that entity.
func (cw *cellWorld) castFrost() uid.UID64 {
	id := cw.brd.CellEntity(cw.target)
	cw.casting = func(cb *goke.CmdBuf) { cw.fx.Cast(cb, id, cw.frost) }
	return id
}

func (cw *cellWorld) kind() board.CellKind { return cw.brd.Res.Logic.Board.Kind(cw.target) }

// The board drops a cell entity itself once its last effect ended, and the terrain keeps what the
// entity's Ground last said — in either order of the two plugins' passes.
func TestCellEntity_GoesWhenItsLastEffectEnds(t *testing.T) {
	for name, boardFirst := range map[string]bool{"effects then board": false, "board then effects": true} {
		t.Run(name, func(t *testing.T) {
			cw := newCellWorld(t, boardFirst)
			first := cw.castFrost()
			cw.ecs.Tick(cellTick) // lands and begins; with the board first, the terrain follows a tick later
			cw.ecs.Tick(cellTick)
			if got := cw.kind(); got != cw.snow {
				t.Fatalf("terrain is %q with the effect on, want snow", got.Name)
			}
			for range 4 {
				cw.ecs.Tick(cellTick)
			}
			if got := cw.kind(); got != cw.grass {
				t.Errorf("terrain is %q after the effect, want grass back", got.Name)
			}
			if again := cw.brd.CellEntity(cw.target); again == first {
				t.Error("the cell entity outlived its last effect; the board should have let it go")
			}
		})
	}
}

// A cast that lands the tick the board finds the entity Idle keeps it: Idle with an Active is not spent.
func TestCellEntity_StaysWhenAnEffectIsCastAgainAsItsLastOneEnded(t *testing.T) {
	cw := newCellWorld(t, true)
	first := cw.castFrost()
	cw.ecs.Tick(cellTick) // lands, begins
	cw.ecs.Tick(cellTick) // running
	cw.ecs.Tick(cellTick) // ends: Active off, Idle on, after the board's pass
	if again := cw.castFrost(); again != first {
		t.Fatalf("the cell entity changed to %d before the board could see it Idle", again)
	}
	cw.ecs.Tick(cellTick) // the cast lands before the board sees Idle beside a fresh Active
	if again := cw.brd.CellEntity(cw.target); again != first {
		t.Errorf("cell entity is %d, want %d kept alive by the new cast", again, first)
	}
	cw.ecs.Tick(cellTick)
	if got := cw.kind(); got != cw.snow {
		t.Errorf("terrain is %q under the renewed frost, want snow", got.Name)
	}
}
