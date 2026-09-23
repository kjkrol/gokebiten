package board_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/game"
	"github.com/kjkrol/gram/internal/engine"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/world"
)

// bodyProbe counts Body entities once the engine's Setup has run.
type bodyProbe struct {
	base  goke.Comp[world.Base]
	marks goke.Comp[plugin.Tags[board.Family]]
	body  plugin.Tag[board.Family]
	query *goke.Query
}

func (p *bodyProbe) SetupSystems() []goke.System {
	return []goke.System{goke.SystemFn{OnInit: func(si *goke.SysInit) {
		p.query = si.NewQueryBuilder(&p.base, &p.marks).Build()
	}}}
}

func (p *bodyProbe) count() int {
	n := 0
	p.query.All()
	for p.query.Next() {
		for _, m := range p.marks.Slice(p.query.Cursor()) {
			if m.Has(p.body) {
				n++
			}
		}
	}
	return n
}

// bodiesStage is a wall column on a board WithCollision, saved or loaded.
type bodiesStage struct {
	loadFrom string
	grid     board.Grid

	world     *world.Plugin
	collision *collision.Plugin
	board     *board.Plugin
	probe     *bodyProbe
	stack     game.Scenes
}

func (g *bodiesStage) Name() string { return "stage" }

func (g *bodiesStage) Init(ctx game.Initializer) error {
	g.world = ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: 6 * cellSize, Height: 16 * cellSize},
		Entities: world.EntitiesCfg{MaxCount: 8, MinSize: unitSize, MaxSize: unitSize},
	})
	g.collision = collision.NewPlugin(g.world)
	if err := ctx.Use(g.collision); err != nil {
		return err
	}
	g.board = board.NewPlugin(g.grid, &board.MultipleOccupancy{}, g.world).WithCollision(g.collision)
	g.board.CellKindDict().Create(
		board.CellKind{Name: "grass", Cost: 1, Allows: board.Land},
		board.CellKind{Name: "wall", Cost: 1, Solid: true},
	)
	if err := ctx.Use(g.board); err != nil {
		return err
	}
	g.probe = &bodyProbe{body: g.board.Body()}
	ctx.Setup(g.probe)
	return nil
}

func (g *bodiesStage) Restore(p game.Persistence) (bool, error) {
	if g.loadFrom == "" {
		return false, nil
	}
	return true, p.Load(g.loadFrom, "")
}

func (g *bodiesStage) Spawn() error {
	var cells []board.CellEntry
	for y := uint32(1); y <= 14; y++ {
		c, _ := g.grid.CellIndex(3, y)
		cells = append(cells, board.CellEntry{Kind: "wall", Cell: c})
	}
	g.board.Seed(board.Layout{Default: "grass", Cells: cells})
	return nil
}

func (g *bodiesStage) Update(ctx goke.RunCtx, d time.Duration) {
	g.world.RunPlan(ctx, d)
	g.collision.RunPlan(ctx, d)
	g.board.RunPlan(ctx, d)
	ctx.Sync()
}

func (g *bodiesStage) Stack() game.Scenes {
	if g.stack == nil {
		g.stack, _ = game.NewStack()
	}
	return g.stack
}

func TestBodies_ALoadRebuildsThemOnceWithoutDuplicates(t *testing.T) {
	path := t.TempDir() + "/save"
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)

	fresh := &bodiesStage{grid: grid}
	eng := engine.NewEngine(oneStageGame{stage: fresh, props: game.Props{}})
	if err := eng.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := fresh.probe.count(); got != 1 {
		t.Fatalf("fresh stage holds %d bodies, want 1", got)
	}
	if err := eng.Persistence().Save(path, ""); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := &bodiesStage{grid: grid, loadFrom: path}
	eng2 := engine.NewEngine(oneStageGame{stage: loaded, props: game.Props{}})
	if err := eng2.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := loaded.probe.count(); got != 1 {
		t.Fatalf("loaded stage holds %d bodies before its first tick, want 1", got)
	}
	for range 3 {
		if err := eng2.Update(); err != nil {
			t.Fatalf("Update: %v", err)
		}
	}
	if got := loaded.probe.count(); got != 1 {
		t.Errorf("loaded stage holds %d bodies after ticking, want 1", got)
	}
	if got := loaded.world.Res.Telemetry.Count; got != 1 {
		t.Errorf("telemetry counts %d entities, want 1", got)
	}
}
