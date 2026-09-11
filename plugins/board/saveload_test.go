package board_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

func testProps() *gokebiten.Props {
	return &gokebiten.Props{World: world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}}
}

type boardSaveLoadTestGame struct {
	worldPlugin *world.Plugin
	boardPlugin *board.Plugin
	grid        board.Grid
	loadFrom    string
}

func (g *boardSaveLoadTestGame) Init(ctx game.Initializer) error {
	g.worldPlugin = ctx.World()
	g.boardPlugin = board.NewPlugin(g.grid, &board.SingleOccupancy{}, nil, g.worldPlugin)
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
func (g *boardSaveLoadTestGame) Spawn() ([]world.Batch, error)                   { return nil, nil }
func (g *boardSaveLoadTestGame) Update(goke.RunCtx, time.Duration)               {}
func (g *boardSaveLoadTestGame) Draw(game.Runtime) []func() render.Renderer      { return nil }
func (g *boardSaveLoadTestGame) HandleEvents(*control.InputEvents, game.Runtime) {}

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

	game := &boardSaveLoadTestGame{grid: grid}
	engine := gokebiten.NewEngine(testProps(), game)
	if err := engine.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	game.boardPlugin.Res.Logic.Board.Set(cell, wall)

	if err := engine.Persistence().Save(basePath, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	game2 := &boardSaveLoadTestGame{grid: grid, loadFrom: basePath}
	engine2 := gokebiten.NewEngine(testProps(), game2)
	if err := engine2.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if got := game2.boardPlugin.Res.Logic.Board.Kind(cell); got != wall {
		t.Errorf("Board().Kind(cell) after Load = %+v, want %+v", got, wall)
	}
}
