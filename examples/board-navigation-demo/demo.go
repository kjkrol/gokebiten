package main

import (
	"image/color"
	"log"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/navigation"
	"github.com/kjkrol/gokebiten/plugins/selection"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/uid"
)

const (
	TPS          = 60
	GridWidth    = 24
	GridHeight   = 16
	CellSize     = 32
	ScreenWidth  = GridWidth * CellSize
	ScreenHeight = GridHeight * CellSize
	EntitySize   = 22
	UnitSpeed    = CellSize * 2
	MaxEntCount  = 10
	hitExpires   = 50 * time.Millisecond

	saveBasePath = "board-navigation-demo"
)

type State struct{ Saves int }

// unit is a roster entry's Data for the "red"/"blue" kinds: where the unit spawns and where it heads.
type unit struct{ start, target board.CellID }

// =========================== Game ===========================

// Demo is the board/navigation/selection demo — exactly one Stage (mainStage below).
type Demo struct{ stage *mainStage }

var _ game.Game = (*Demo)(nil)

func NewDemo() *Demo { return &Demo{stage: &mainStage{}} }

func (d *Demo) Props() game.Props {
	return game.Props{
		Title:       "gokebiten board & navigation plugins demo",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	}
}

func (d *Demo) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{d.stage.Name(): d.stage}, d.stage.Name()
}

// =========================== Stage ===========================

// mainStage wires the board/navigation/selection demo — its plugins are its own fields, built in Init.
type mainStage struct {
	world      *world.Plugin
	board      *board.Plugin
	nav        *navigation.Plugin
	collisions *collisions.Plugin
	selection  *selection.Plugin
	stack      game.Stack
	state      *State
}

var _ game.Stage = (*mainStage)(nil)

func (s *mainStage) Name() string { return "board-navigation-demo" }

func (s *mainStage) Stack() game.Stack { return s.stack }

func (s *mainStage) Init(ctx game.Initializer) error {
	s.world = ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: false},
		Entities: world.EntitiesCfg{MaxCount: MaxEntCount, MinSize: EntitySize, MaxSize: EntitySize},
	})
	s.world.WithCameraControls()

	grid := board.DefaultGrids{}.Square(GridWidth, GridHeight, CellSize)
	s.board = board.NewPlugin(grid, &board.SingleOccupancy{}, s.world)
	s.registerCellKinds()
	if err := ctx.Use(s.board); err != nil {
		return err
	}

	s.nav = navigation.NewPlugin(UnitSpeed, s.board, s.world)
	if err := ctx.Use(s.nav); err != nil {
		return err
	}

	s.collisions = collisions.NewPlugin(hitExpires, s.world)
	if err := ctx.Use(s.collisions); err != nil {
		return err
	}

	s.selection = selection.NewPlugin(s.world)
	s.state = &State{}
	if err := ctx.Use(s.selection); err != nil {
		return err
	}

	s.registerUnitKinds()

	main := &mainScene{stage: s}
	stack, err := game.NewStack(main)
	if err != nil {
		return err
	}
	s.stack = stack
	comp := stack.Composition()
	comp.Show(main.Name())
	return ctx.Track(comp)
}

// registerCellKinds defines every terrain kind the board can hold.
func (s *mainStage) registerCellKinds() {
	s.board.CellKindDict().Create(
		board.CellKind{Name: "grass", Cost: 1, Passable: true},
		board.CellKind{Name: "wall", Cost: 1, Passable: false},
		board.CellKind{Name: "road", Cost: 0.4, Passable: true},
	)
}

// registerUnitKinds defines the "red"/"blue" unit kinds, each spawned from a unit roster entry.
func (s *mainStage) registerUnitKinds() {
	brd := s.board.Res.Logic.Board
	occupancy := s.board.Occupancy()
	unitKind := func(name string) world.EntKind {
		return world.EntKind{
			Name:     name,
			Position: world.Load(func(u unit) world.Position { return world.Position{AABB: board.CellAABB(brd, u.start, EntitySize)} }),
			Velocity: world.Const(world.Velocity{}),
			Components: []world.ComponentTemplate{
				world.Load(func(u unit) navigation.MoveOrder { return navigation.MoveOrder{Target: u.target} }),
				world.Load(func(u unit) board.Cell { return board.Cell{ID: u.start} }).
					WithEffect(func(c board.Cell, id uid.UID64) { occupancy.Enter(c.ID, id) }),
				world.Const(selection.Selected{}),
				world.Const(collisions.Collision{}),
			},
		}
	}
	s.world.EntKindDict().Create(unitKind("red"), unitKind("blue"))
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
	log.Printf("loaded saved board (save #%d)", s.state.Saves)
	return true, nil
}

func (s *mainStage) Spawn() error {
	brd := s.board.Res.Logic.Board
	cell := func(x, y uint32) board.CellID { c, _ := brd.CellIndex(x, y); return c }

	var walls []board.CellEntry
	for y := uint32(2); y < GridHeight; y++ {
		walls = append(walls, board.CellEntry{Kind: "wall", Cell: cell(wallCol, y)})
	}
	s.board.Seed(board.Layout{Default: "grass", Cells: walls})

	s.world.Seed(world.Roster{
		{Kind: "red", Data: unit{start: cell(2, 4), target: cell(GridWidth-3, 4)}},
		{Kind: "blue", Data: unit{start: cell(2, 12), target: cell(GridWidth-3, 12)}},
	})
	return nil
}

func (s *mainStage) Update(ctx goke.RunCtx, d time.Duration) {
	s.world.RunPlan(ctx, d)
	s.collisions.RunPlan(ctx, d)
	s.nav.RunPlan(ctx, d)
	s.selection.RunPlan(ctx, d)
	ctx.Sync()
}

// =========================== Scene ===========================

type mainScene struct{ stage *mainStage }

var _ game.Scene = (*mainScene)(nil)

func (m *mainScene) Name() string { return "main" }

func (m *mainScene) Layers() []func() render.Renderer {
	s := m.stage

	entKinds := s.world.EntKindDict().All()
	palette := map[string]color.RGBA{
		"red":  {R: 220, G: 90, B: 90, A: 255},
		"blue": {R: 90, G: 140, B: 220, A: 255},
	}
	worldAtlas := render.NewAtlas(EntitySize, len(entKinds))
	for _, kind := range entKinds {
		worldAtlas.RegisterAt(kind.SpriteID, render.Solid(palette[kind.Name]))
	}
	worldAtlas.Close()
	s.world.WithRenderer(worldAtlas)

	kinds := s.board.CellKindDict()
	grass, _ := kinds.Get("grass")
	wall, _ := kinds.Get("wall")
	road, _ := kinds.Get("road")
	boardAtlas := render.NewAtlas(CellSize, len(kinds.All()))
	boardAtlas.RegisterAt(grass.SpriteID, render.Solid(color.RGBA{R: 60, G: 95, B: 60, A: 255}))
	boardAtlas.RegisterAt(wall.SpriteID, render.Solid(color.RGBA{R: 40, G: 40, B: 40, A: 255}))
	boardAtlas.RegisterAt(road.SpriteID, render.Solid(color.RGBA{R: 150, G: 130, B: 80, A: 255}))
	boardAtlas.Close()
	s.board.WithRenderer(boardAtlas)

	pathAtlas, pathSprites := navigation.RegisterDefaultPathSprites(CellSize, 2, color.RGBA{R: 255, G: 140, B: 0, A: 255})
	s.nav.SetPathSprites(pathSprites)
	s.nav.WithRenderer(pathAtlas)

	s.selection.WithRenderer(nil)

	return []func() render.Renderer{s.board.Renderer, s.nav.Renderer, s.world.Renderer, s.selection.Renderer}
}

func (m *mainScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, composition game.Composition) {
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
		case ebiten.KeyR:
			buildShortcut(s.board.Res.Logic.Board, s.board.CellKindDict())
			log.Print("built a road through the wall — in-flight units re-path onto it as soon as they deviate")
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

const (
	wallCol     = 12
	shortcutRow = 8
)

func buildShortcut(brd *board.Board, kinds board.CellKindDict) {
	road, _ := kinds.Get("road")
	c, _ := brd.CellIndex(wallCol, shortcutRow)
	brd.Set(c, road)
}
