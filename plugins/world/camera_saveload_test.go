package world_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/internal/engine"
	"github.com/kjkrol/gokebiten/plugins/world"
)

func testEngineProps() game.Props {
	return game.Props{World: world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	}}
}

type cameraSaveLoadTestGame struct {
	world    *world.Plugin
	loadFrom string

	stack game.Stack
}

func (g *cameraSaveLoadTestGame) Name() string { return "stage" }
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
func (g *cameraSaveLoadTestGame) Spawn() error                      { return nil }
func (g *cameraSaveLoadTestGame) Update(goke.RunCtx, time.Duration) {}
func (g *cameraSaveLoadTestGame) Stack() game.Stack {
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

// TestPlugin_SaveLoad_CameraRoundTrip guards that the shared Camera's
// Viewport/Zoom is saved/restored automatically via world's Serializable,
// without the caller ever passing it to Persistence.Save/Load explicitly.
func TestPlugin_SaveLoad_CameraRoundTrip(t *testing.T) {
	basePath := t.TempDir() + "/save"

	g := &cameraSaveLoadTestGame{}
	eng := engine.NewEngine(oneStageGame{stage: g, props: testEngineProps()})
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
	eng2 := engine.NewEngine(oneStageGame{stage: g2, props: testEngineProps()})
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
