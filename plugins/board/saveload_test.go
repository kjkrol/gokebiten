package board_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
)

func testProps() game.Props {
	return game.Props{World: world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}}
}

type boardSaveLoadTestGame struct {
	worldPlugin *world.Plugin
	boardPlugin *board.Plugin
	grid        board.Grid
	loadFrom    string

	stack game.Stack
}

func (g *boardSaveLoadTestGame) Name() string { return "stage" }
func (g *boardSaveLoadTestGame) Init(ctx game.Initializer) error {
	g.worldPlugin = ctx.World()
	g.boardPlugin = board.NewPlugin(g.grid, &board.SingleOccupancy{}, g.worldPlugin)
	return ctx.Use(g.boardPlugin)
}
func (g *boardSaveLoadTestGame) Restore(p game.Persistence) (bool, error) {
	if g.loadFrom == "" {
		return false, nil
	}
	if err := p.Load(g.loadFrom, ""); err != nil {
		return false, err
	}
	return true, nil
}
func (g *boardSaveLoadTestGame) Spawn() error                      { return nil }
func (g *boardSaveLoadTestGame) Update(goke.RunCtx, time.Duration) {}
func (g *boardSaveLoadTestGame) Stack() game.Stack {
	if g.stack == nil {
		g.stack, _ = game.NewStack()
	}
	return g.stack
}

// oneStageGame is a minimal game.Game wrapping a single Stage.
type oneStageGame struct {
	stage game.Stage
	props game.Props
}

func (g oneStageGame) Props() game.Props { return g.props }

func (g oneStageGame) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{g.stage.Name(): g.stage}, g.stage.Name()
}

// TestPlugin_SaveLoad_TerrainRoundTrip guards that board.Resources' TerrainMap
// is saved/restored via Serializable, without the caller ever passing it to
// Persistence.Save/Load explicitly.
func TestPlugin_SaveLoad_TerrainRoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"
	grid := board.DefaultGrids{}.Square(5, 5, 10)
	wall := board.CellKind{Name: "wall", Cost: 1, Passable: false}
	cell, ok := grid.CellAt(geom.NewVec(21.0, 21.0))
	if !ok {
		t.Fatal("expected (21,21) to land inside the 5x5 grid")
	}

	stage := &boardSaveLoadTestGame{grid: grid}
	eng := engine.NewEngine(oneStageGame{stage: stage, props: testProps()})
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	stage.boardPlugin.Res.Logic.Board.Set(cell, wall)

	if err := eng.Persistence().Save(basePath, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	game2 := &boardSaveLoadTestGame{grid: grid, loadFrom: basePath}
	eng2 := engine.NewEngine(oneStageGame{stage: game2, props: testProps()})
	if err := eng2.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if got := game2.boardPlugin.Res.Logic.Board.Kind(cell); got != wall {
		t.Errorf("Board().Kind(cell) after Load = %+v, want %+v", got, wall)
	}
}
