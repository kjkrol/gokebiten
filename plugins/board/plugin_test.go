package board

import (
	"testing"

	"github.com/kjkrol/gokebiten/plugins/world"
)

func TestNewPlugin_SetsGridToroidalFromWorldConfig(t *testing.T) {
	grid := DefaultGrids{}.Square(5, 5, 10).(*squareGrid)
	if grid.Toroidal {
		t.Fatal("expected DefaultGrids.Square to default Toroidal=false")
	}

	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 100, Height: 100, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	NewPlugin(grid, &SingleOccupancy{}, nil, worldPlugin)

	if !grid.Toroidal {
		t.Error("expected NewPlugin to set grid.Toroidal to match world.Config.Space.Toroidal=true")
	}
}
