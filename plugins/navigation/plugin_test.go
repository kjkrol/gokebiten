package navigation

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// stubInstallCtx is a minimal plugin.Installer for tests that call Install directly.
type stubInstallCtx struct {
	ecs     *goke.ECS
	pending []func() []goke.System
}

func (c *stubInstallCtx) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(si *goke.SysInit) { m.RegSystems(c.ecs) }}
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}
func (c *stubInstallCtx) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.pending = append(c.pending, p.SetupSystems)
	}
}
func (c *stubInstallCtx) RegSys(factory func() goke.System) goke.Runnable {
	return c.ecs.RegSys(factory())
}
func (c *stubInstallCtx) ECS() *goke.ECS { return c.ecs }

// TestPlugin_Install_WiresBoardForEventHandler guards against the exact
// regression reported live: Install fetching *board.Board into a local
// var but never assigning it to Plugin.board, leaving EventHandler's
// DefaultCommandEventHandler holding a nil Grid that panics on the first
// right-click.
func TestPlugin_Install_WiresBoardForEventHandler(t *testing.T) {
	grid := board.DefaultGrids{}.Square(5, 5, 10)
	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 50, Height: 50},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	boardPlugin := board.NewPlugin(grid, &board.SingleOccupancy{}, nil, worldPlugin)
	boardPlugin.Res.Logic.Board.SetAll(board.CellKind{Cost: 1, Passable: true})

	surface := plane.NewEuclidean2D[uint32](50, 50)
	cam := camera.NewBasicCamera(surface, geom.NewAABBAt(geom.NewVec[uint32](0, 0), 50, 50))

	navPlugin := NewPlugin(10, boardPlugin, worldPlugin, cam)
	ctx := &stubInstallCtx{ecs: goke.New()}
	if err := navPlugin.Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}

	events := &control.InputEvents{}
	events.AddClickEvent(25, 25, ebiten.MouseButtonRight, control.ActionPress)

	navPlugin.EventHandler().HandleEvents(events) // must not panic — this is the exact crash site

	if navPlugin.res.PendingTarget == nil {
		t.Error("expected a right-click to set Resources.PendingTarget")
	}
}
