package navigation

import (
	"github.com/kjkrol/gram/plugins/selection"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/players"
	"github.com/kjkrol/gram/plugins/world"
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

func TestPlugin_DefaultBindings_TurnARightClickIntoMoveTo(t *testing.T) {
	grid := board.DefaultGrids{}.Square(5, 5, 10)
	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 50, Height: 50},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	boardPlugin := board.NewPlugin(grid, &board.SingleOccupancy{}, worldPlugin)
	boardPlugin.Res.Logic.Board.SetAll(board.CellKind{Cost: 1, Allows: board.Land})
	pl := players.NewPlugin(worldPlugin)
	navPlugin := NewPlugin(boardPlugin, worldPlugin, selection.NewPlugin(worldPlugin, pl), pl)
	local := pl.Local("tester")
	if err := local.Bind(navPlugin.DefaultBindings()...); err != nil {
		t.Fatal(err)
	}
	want, _ := grid.CellIndex(2, 2)

	events := &control.InputEvents{}
	events.AddClickEvent(25, 25, ebiten.MouseButtonRight, control.ActionPress)
	pl.EventHandler().HandleEvents(events)
	var got []MoveTo
	navPlugin.moves.Drain(func(i players.Issued[MoveTo]) { got = append(got, i.Command) })
	if len(got) != 1 || got[0] != (MoveTo{Cell: want}) {
		t.Errorf("a right click issued %v, want one MoveTo to %v", got, want)
	}

	events = &control.InputEvents{}
	events.Modifiers.Shift = true
	events.AddClickEvent(25, 25, ebiten.MouseButtonRight, control.ActionPress)
	pl.EventHandler().HandleEvents(events)
	got = got[:0]
	navPlugin.moves.Drain(func(i players.Issued[MoveTo]) { got = append(got, i.Command) })
	if len(got) != 1 || !got[0].Append {
		t.Errorf("a Shift right click issued %v, want one MoveTo that appends", got)
	}
}
