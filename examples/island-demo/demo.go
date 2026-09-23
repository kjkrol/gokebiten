// Command island-demo is a map larger than the window: an island of fields, forests, slow hills
// and slower mountains in a sea that drowns whoever is pushed in, a road round it, units under
// orders. Scroll with the wheel, drag with the
// middle button or push the cursor to an edge to move the camera.
package main

import (
	"image/color"
	"log"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/game"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/navigation"
	"github.com/kjkrol/gram/plugins/selection"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/gram/render"
	"github.com/kjkrol/uid"
)

const (
	TPS          = 60
	GridWidth    = 96
	GridHeight   = 64
	CellSize     = 32
	WorldWidth   = GridWidth * CellSize
	WorldHeight  = GridHeight * CellSize
	ScreenWidth  = 1024
	ScreenHeight = 768
	EntitySize   = 22
	UnitSpeed    = CellSize * 3
	UnitCount    = 6
	// MaxEntCount is the units plus the terrain bodies the forests make.
	MaxEntCount = 400

	saveBasePath = "island-demo"
)

type State struct{ Saves int }

// =========================== Game ===========================

// Demo is the island demo — exactly one Stage (mainStage below).
type Demo struct{ stage *mainStage }

var _ game.Game = (*Demo)(nil)

func NewDemo() *Demo { return &Demo{stage: &mainStage{}} }

func (d *Demo) Props() game.Props {
	return game.Props{
		Title:       "gram — an island under a moving camera",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	}
}

func (d *Demo) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{d.stage.Name(): d.stage}, d.stage.Name()
}

// =========================== Stage ===========================

type mainStage struct {
	world     *world.Plugin
	board     *board.Plugin
	nav       *navigation.Plugin
	collision *collision.Plugin
	selection *selection.Plugin
	unit      kind.Of[unit]
	stack     game.Scenes
	state     *State
}

var _ game.Stage = (*mainStage)(nil)

func (s *mainStage) Name() string { return "island-demo" }

func (s *mainStage) Stack() game.Scenes { return s.stack }

func (s *mainStage) Init(ctx game.Initializer) error {
	s.world = ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: WorldWidth, Height: WorldHeight},
		Entities: world.EntitiesCfg{MaxCount: MaxEntCount, MinSize: EntitySize, MaxSize: EntitySize},
	})
	s.world.WithCameraControls()

	s.collision = collision.NewPlugin(s.world)
	if err := ctx.Use(s.collision); err != nil {
		return err
	}

	grid := board.DefaultGrids{}.Square(GridWidth, GridHeight, CellSize)
	s.board = board.NewPlugin(grid, &board.MultipleOccupancy{}, s.world).WithCollision(s.collision)
	s.board.CellKindDict().Create(
		board.CellKind{Name: "water", Cost: 1, Allows: board.Water},
		board.CellKind{Name: "field", Cost: 1.5, Allows: board.Land},
		board.CellKind{Name: "forest", Cost: 3, Allows: board.Land, Opaque: true},
		board.CellKind{Name: "hills", Cost: 4, Allows: board.Land},
		board.CellKind{Name: "mountain", Cost: 8, Allows: board.Land},
		board.CellKind{Name: "road", Cost: 1, Allows: board.Land},
	)
	if err := s.board.RegisterBehavior(plugin.Each[board.Mover](s.drown)); err != nil {
		return err
	}
	if err := ctx.Use(s.board); err != nil {
		return err
	}

	s.selection = selection.NewPlugin(s.world)
	if err := ctx.Use(s.selection); err != nil {
		return err
	}

	s.nav = navigation.NewPlugin(s.board, s.world, s.selection)
	if err := ctx.Use(s.nav); err != nil {
		return err
	}

	s.state = &State{}

	s.defineKinds()

	main := &mainScene{stage: s, tps: ctx.TPS()}
	stack, err := game.NewStack(main)
	if err != nil {
		return err
	}
	s.stack = stack
	comp := stack.Composition()
	comp.Show(main.Name())
	return ctx.Track(comp)
}

func (s *mainStage) Restore(p game.Persistence) (bool, error) {
	saves, err := p.List(saveBasePath)
	if err != nil {
		return false, err
	}
	if !slices.Contains(saves, "") {
		return false, nil
	}
	if err := p.Load(saveBasePath, "", s.state); err != nil {
		return false, err
	}
	log.Printf("loaded saved island (save #%d)", s.state.Saves)
	return true, nil
}

// unit is the row the unit kind spawns from: where it starts and where it heads.
type unit struct{ start, target board.CellID }

// defineKinds says what this game's entities are, fresh or restored.
func (s *mainStage) defineKinds() {
	brd := s.board.Res.Logic.Board
	occupancy := s.board.Occupancy()
	s.unit = kind.Define[unit](s.world.Kinds(), "unit", kind.Spec{
		kind.Load(func(u unit) world.Position { return world.Position{AABB: board.CellAABB(brd, u.start, EntitySize)} }),
		kind.Const(world.Velocity{}),
		kind.Const(world.Steering{MaxSpeed: UnitSpeed, Accel: UnitSpeed * 2, Brake: UnitSpeed * 4, V0: UnitSpeed / 2, TurnRate: 0.15}),
		kind.Load(func(u unit) navigation.MoveOrder { return navigation.MoveOrder{Target: u.target} }),
		kind.Load(func(u unit) board.Cell { return board.Cell{ID: u.start} }).
			WithEffect(func(c board.Cell, id uid.UID64) { occupancy.Enter(c.ID, id) }),
		kind.Tagged(s.selection.Tags().Selectable, s.selection.Tags().Selected),
		kind.Const(collision.Collider{}),
		kind.Const(collision.Physics{}),
		kind.Const(board.Mover{Domain: board.Land}),
	})
}

// Spawn lays the island out and puts a unit at every road stop, bound for the opposite one.
func (s *mainStage) Spawn() error {
	layout, stops := islandLayout(s.board.Res.Logic.Board)
	s.board.Seed(layout)

	entries := make([]kind.Entry, 0, len(stops))
	for i, from := range stops {
		entries = append(entries, s.unit.Entry(unit{start: from, target: stops[(i+len(stops)/2)%len(stops)]}))
	}
	s.world.Seed(entries...)
	return nil
}

// drown despawns a unit standing where its domain may not — pushed into the sea, say.
func (s *mainStage) drown(t plugin.Tick, m *board.Mover, st board.Standing) {
	if st.Fell(m.Domain) {
		log.Printf("unit %d drowned in the %s at cell %d", st.ID, st.Kind.Name, st.Cell)
		s.world.Despawn(t.CmdBuf, st.ID)
	}
}

func (s *mainStage) Update(ctx goke.RunCtx, d time.Duration) {
	s.world.RunPlan(ctx, d)
	s.collision.RunPlan(ctx, d)
	s.board.RunPlan(ctx, d)
	s.nav.RunPlan(ctx, d)
	s.selection.RunPlan(ctx, d)
	ctx.Sync()
}

// =========================== Scene ===========================

type mainScene struct {
	stage *mainStage
	tps   *game.TPS
	none  int
}

var _ game.Scene = (*mainScene)(nil)

func (m *mainScene) Name() string { return "main" }

func (m *mainScene) Layers() []render.Renderer {
	s := m.stage

	worldAtlas := render.NewAtlas()
	worldAtlas.RegisterAt(s.unit.SpriteID(), EntitySize, render.Solid(color.RGBA{R: 230, G: 80, B: 80, A: 255}))
	worldAtlas.Close()
	s.world.WithRenderer(worldAtlas)

	kinds := s.board.CellKindDict()
	boardAtlas := render.NewAtlas()
	for name, c := range map[string]color.RGBA{
		"water":    {R: 40, G: 90, B: 170, A: 255},
		"field":    {R: 120, G: 170, B: 80, A: 255},
		"forest":   {R: 30, G: 90, B: 45, A: 255},
		"hills":    {R: 150, G: 140, B: 70, A: 255},
		"mountain": {R: 120, G: 120, B: 125, A: 255},
		"road":     {R: 190, G: 170, B: 120, A: 255},
	} {
		k, _ := kinds.Get(name)
		boardAtlas.RegisterAt(k.SpriteID, CellSize, render.Solid(c))
	}
	boardAtlas.Close()
	s.board.WithRenderer(boardAtlas)
	s.board.Res.Render.ShowGridLines = false

	pathAtlas, pathSprites := navigation.RegisterDefaultPathSprites(CellSize, 2, color.RGBA{R: 255, G: 140, B: 0, A: 255})
	s.nav.SetPathSprites(pathSprites)
	s.nav.WithRenderer(pathAtlas)
	s.selection.WithRenderer(nil)

	count := func() int { return s.world.Res.Telemetry.Count }
	return []render.Renderer{
		s.board.Renderer(), s.nav.Renderer(), s.world.Renderer(), s.selection.Renderer(),
		render.NewTelemetryRenderer(&m.tps.Ticks, count, &m.none),
	}
}

func (m *mainScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, _ game.Composition) {
	s := m.stage
	s.selection.EventHandler().HandleEvents(events)
	s.nav.EventHandler().HandleEvents(events)
	s.world.EventHandler().HandleEvents(events)
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeyEscape:
			runtime.Quit()
		case ebiten.KeySpace:
			runtime.TogglePause()
		case ebiten.KeyB:
			s.board.Res.Render.ToggleShowGridLines()
		case ebiten.KeyF5:
			s.state.Saves++
			if err := runtime.Persistence().Save(saveBasePath, "", s.state); err != nil {
				log.Printf("save: %v", err)
				continue
			}
			log.Printf("saved (save #%d)", s.state.Saves)
		}
	}
}

func (m *mainScene) Focusable() bool { return true }
