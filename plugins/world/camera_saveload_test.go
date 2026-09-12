package world_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

func testEngineProps() *engine.Props {
	return &engine.Props{World: world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}}
}

type cameraSaveLoadTestGame struct {
	world    *world.Plugin
	loadFrom string
}

func (g *cameraSaveLoadTestGame) Init(ctx game.Initializer) error {
	g.world = ctx.World()
	return nil
}
func (g *cameraSaveLoadTestGame) Restore(p game.Persistence) (bool, error) {
	if g.loadFrom == "" {
		return false, nil
	}
	if err := p.Load(g.loadFrom, ""); err != nil {
		return false, err
	}
	return true, nil
}
func (g *cameraSaveLoadTestGame) Spawn() ([]world.Batch, error)                   { return nil, nil }
func (g *cameraSaveLoadTestGame) Update(goke.RunCtx, time.Duration)               {}
func (g *cameraSaveLoadTestGame) Draw(game.Runtime) []func() render.Renderer      { return nil }
func (g *cameraSaveLoadTestGame) HandleEvents(*control.InputEvents, game.Runtime) {}

// TestPlugin_SaveLoad_CameraRoundTrip guards that the shared Camera's
// Viewport/Zoom is saved/restored automatically via world's Serializable,
// without the caller ever passing it to Persistence.Save/Load explicitly.
func TestPlugin_SaveLoad_CameraRoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"

	g := &cameraSaveLoadTestGame{}
	eng := engine.NewEngine(testEngineProps(), g)
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	g.world.Camera().MoveTo(100, 150)
	b := g.world.Camera().Bounds()
	cx, cy := float32(b.TopLeft.X+b.BottomRight.X)/2, float32(b.TopLeft.Y+b.BottomRight.Y)/2
	g.world.Camera().ZoomIn(2, cx, cy)
	wantBounds := g.world.Camera().Bounds()
	wantZoom := g.world.Camera().Zoom()

	if err := eng.Persistence().Save(basePath, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	g2 := &cameraSaveLoadTestGame{loadFrom: basePath}
	eng2 := engine.NewEngine(testEngineProps(), g2)
	if err := eng2.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	if got := g2.world.Camera().Bounds(); got != wantBounds {
		t.Errorf("Camera().Bounds() after Load = %+v, want %+v", got, wantBounds)
	}
	if got := g2.world.Camera().Zoom(); got != wantZoom {
		t.Errorf("Camera().Zoom() after Load = %v, want %v", got, wantZoom)
	}
}
