package main

import (
	"image/color"
	"log"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/board"
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

	saveBasePath = "board-navigation-demo"
)

type State struct{ Saves int }

// Demo wires the board/navigation/selection demo — its plugins are its own fields, built in Init.
type Demo struct {
	world     *world.Plugin
	board     *board.Plugin
	nav       *navigation.Plugin
	selection *selection.Plugin

	state *State
}

var _ game.Game = (*Demo)(nil)

func (dm *Demo) Init(ctx game.Initializer) error {
	worldAtlas := render.NewAtlas(EntitySize, 2)
	redSprite := worldAtlas.Register(render.Solid(color.RGBA{R: 220, G: 90, B: 90, A: 255}))
	blueSprite := worldAtlas.Register(render.Solid(color.RGBA{R: 90, G: 140, B: 220, A: 255}))
	worldAtlas.Close()

	worldCfg := world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: false},
		Entities: world.EntitiesCfg{MaxCount: MaxEntCount, MinSize: EntitySize, MaxSize: EntitySize},
	}
	dm.world = world.NewPlugin(worldCfg)
	cam := camera.NewFromSpace(worldCfg.Space.Width, worldCfg.Space.Height, worldCfg.Space.Toroidal)
	dm.world.WithRenderer(cam, worldAtlas)
	if err := ctx.Use(dm.world); err != nil {
		return err
	}

	boardAtlas := render.NewAtlas(CellSize, 3)
	grassSprite := boardAtlas.Register(render.Solid(color.RGBA{R: 60, G: 95, B: 60, A: 255}))
	wallSprite := boardAtlas.Register(render.Solid(color.RGBA{R: 40, G: 40, B: 40, A: 255}))
	roadSprite := boardAtlas.Register(render.Solid(color.RGBA{R: 150, G: 130, B: 80, A: 255}))
	boardAtlas.Close()
	kinds := board.NewCellKindDict(
		board.CellKind{Name: "grass", Cost: 1, Passable: true, SpriteID: grassSprite},
		board.CellKind{Name: "wall", Cost: 1, Passable: false, SpriteID: wallSprite},
		board.CellKind{Name: "road", Cost: 0.4, Passable: true, SpriteID: roadSprite},
	)
	grid := board.DefaultGrids{}.Square(GridWidth, GridHeight, CellSize)
	occupancy := &board.SingleOccupancy{}
	dm.board = board.NewPlugin(grid, occupancy, kinds, dm.world)
	dm.board.WithRenderer(cam, boardAtlas)
	if err := ctx.Use(dm.board); err != nil {
		return err
	}

	pathAtlas, pathSprites := navigation.RegisterDefaultPathSprites(CellSize, 2, color.RGBA{R: 255, G: 140, B: 0, A: 255})
	dm.nav = navigation.NewPlugin(UnitSpeed, dm.board, dm.world, cam)
	dm.nav.SetPathSprites(pathSprites) // TODO: try do this better
	dm.nav.WithRenderer(cam, pathAtlas)
	if err := ctx.Use(dm.nav); err != nil {
		return err
	}

	dm.selection = selection.NewPlugin(dm.world, cam)
	dm.selection.WithRenderer(cam, nil)
	if err := ctx.Use(dm.selection); err != nil {
		return err
	}

	saves, err := ctx.Runtime().Persistence().List(saveBasePath)
	if err != nil {
		return err
	}
	dm.state = &State{}
	if slices.Contains(saves, "") {
		if err := ctx.Runtime().Persistence().Load(saveBasePath, "", dm.state); err != nil {
			return err
		}
		log.Printf("loaded saved board (save #%d)", dm.state.Saves)
		return nil
	}

	brd := dm.board.Res.Logic.Board
	brd.SetAll(kinds["grass"])
	buildWall(brd, kinds)

	unitRoster := [...]struct {
		startX, startY uint32
		targetX        uint32
		sprite         render.SpriteID
	}{
		{startX: 2, startY: 4, targetX: GridWidth - 3, sprite: redSprite},
		{startX: 2, startY: 12, targetX: GridWidth - 3, sprite: blueSprite},
	}
	spawner := world.NewSpawner(
		func(index, count int) world.Position {
			spawn := unitRoster[index]
			start, _ := brd.CellIndex(spawn.startX, spawn.startY)
			return world.Position{AABB: board.CellAABB(brd, start, EntitySize)}
		},
		func(index int) world.Velocity { return world.Velocity{} },
	).
		WithEffect(func(index int) board.Cell {
			spawn := unitRoster[index]
			c, _ := brd.CellIndex(spawn.startX, spawn.startY)
			return board.Cell{ID: c}
		}, func(c board.Cell, id uid.UID64) {
			occupancy.Enter(c.ID, id)
		}).
		With(func(index int) navigation.MoveOrder {
			spawn := unitRoster[index]
			target, _ := brd.CellIndex(spawn.targetX, spawn.startY)
			return navigation.MoveOrder{Target: target}
		}).
		With(func(index int) world.Appearance {
			return world.Appearance{SpriteID: unitRoster[index].sprite}
		}).
		With(func(index int) selection.Selected { return selection.Selected{} })
	dm.world.Populate(len(unitRoster), spawner)
	return nil
}

func (dm *Demo) RunPlan(ctx goke.RunCtx, d time.Duration) {
	dm.world.RunPlan(ctx, d)
	dm.nav.RunPlan(ctx, d)
	dm.selection.RunPlan(ctx, d)
	ctx.Sync()
}

func (dm *Demo) Layers(runtime game.Runtime) []func() render.Renderer {
	return []func() render.Renderer{dm.board.Renderer, dm.nav.Renderer, dm.world.Renderer, dm.selection.Renderer}
}

func (dm *Demo) HandleEvents(events *control.InputEvents, runtime game.Runtime) {
	dm.selection.EventHandler().HandleEvents(events)
	dm.nav.EventHandler().HandleEvents(events)
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeySpace:
			runtime.TogglePause()
		case ebiten.KeyB:
			dm.board.Res.Render.ShowGridLines = !dm.board.Res.Render.ShowGridLines
		case ebiten.KeyR:
			buildShortcut(dm.board.Res.Logic.Board, dm.board.Res.Logic.Kinds)
			log.Print("built a road through the wall — in-flight units re-path onto it as soon as they deviate")
		case ebiten.KeyF5:
			dm.state.Saves++
			if err := runtime.Persistence().Save(saveBasePath, "", dm.state); err != nil {
				log.Printf("save: %v", err)
				continue
			}
			log.Printf("saved (save #%d)", dm.state.Saves)
		}
	}
}

const (
	wallCol     = 12
	shortcutRow = 8
)

func buildWall(brd *board.Board, kinds board.CellKindDict) {
	for y := uint32(2); y < GridHeight; y++ {
		c, _ := brd.CellIndex(wallCol, y)
		brd.Set(c, kinds["wall"])
	}
}

func buildShortcut(brd *board.Board, kinds board.CellKindDict) {
	c, _ := brd.CellIndex(wallCol, shortcutRow)
	brd.Set(c, kinds["road"])
}
