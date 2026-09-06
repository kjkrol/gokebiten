package navigation

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// TestPlugin_Install_WiresBoardForEventHandler guards against the exact
// regression reported live: Install fetching *board.Board into a local
// var but never assigning it to Plugin.board, leaving EventHandler's
// DefaultCommandEventHandler holding a nil Grid that panics on the first
// right-click.
func TestPlugin_Install_WiresBoardForEventHandler(t *testing.T) {
	grid := board.DefaultGrids{}.Square(5, 5, 10)
	brd := board.NewBoard(grid, board.NewTerrainMap())
	brd.SetAll(board.CellKind{Cost: 1, Passable: true})

	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 50, Height: 50},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	boardPlugin := board.NewPlugin(grid, &board.SingleOccupancy{}, nil, worldPlugin)

	surface := plane.NewEuclidean2D[uint32](50, 50)
	camera := render.NewBasicCamera(surface, geom.NewAABBAt(geom.NewVec[uint32](0, 0), 50, 50))

	resources := plugins.NewResources()
	resources.Insert(brd)
	resources.Insert[render.Camera](camera)
	ctx := plugins.NewGameCtx(resources, goke.New(),
		func(any) {}, func(func() []goke.System) {}, func(string) bool { return true })

	navPlugin := NewPlugin(10, boardPlugin, worldPlugin)
	if err := navPlugin.Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	events := &control.InputEvents{}
	events.AddClickEvent(25, 25, ebiten.MouseButtonRight, control.ActionPress)

	navPlugin.EventHandler().HandleEvents(events) // must not panic — this is the exact crash site

	if navPlugin.commandState.PendingTarget == nil {
		t.Error("expected a right-click to set CommandState.PendingTarget")
	}
}
