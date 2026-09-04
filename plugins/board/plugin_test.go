package board

import (
	"testing"

	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestPlugin_Install_SetsGridToroidalFromWorldConfig(t *testing.T) {
	grid := DefaultGrids{}.Square(5, 5, 10).(*squareGrid)
	if grid.Toroidal {
		t.Fatal("expected DefaultGrids.Square to default Toroidal=false")
	}

	game := gokebiten.NewGame(&gokebiten.GameProps{})
	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	if err := game.UsePlugin(worldPlugin); err != nil {
		t.Fatalf("UsePlugin(world): %v", err)
	}
	boardPlugin := NewPlugin(grid, &SingleOccupancy{}, nil, worldPlugin)
	if err := game.UsePlugin(boardPlugin); err != nil {
		t.Fatalf("UsePlugin(board): %v", err)
	}

	if !grid.Toroidal {
		t.Error("expected Plugin.Install to set grid.Toroidal to match world.Config.Space.Toroidal=true")
	}
}
