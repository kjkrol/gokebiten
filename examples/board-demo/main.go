package main

import (
	"image/color"
	"log"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/camera"
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

	saveBasePath = "board-rts-demo"
)

var (
	Grass = board.CellKind{Name: "grass", Cost: 1, Passable: true}
	Wall  = board.CellKind{Name: "wall", Cost: 1, Passable: false}
	Road  = board.CellKind{Name: "road", Cost: 0.4, Passable: true}
)

type State struct{ Saves int }

func main() {
	game := gokebiten.NewGame(&gokebiten.GameProps{
		Title:       "gokebiten board plugin — square grid RTS demo",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	})

	grid := board.NewSquareGrid(GridWidth, GridHeight, CellSize)
	occupancy := &board.SingleOccupancy{}

	atlas := render.NewAtlas(16, 8)
	grassSprite := atlas.Register(render.Solid(color.RGBA{R: 60, G: 95, B: 60, A: 255}))
	wallSprite := atlas.Register(render.Solid(color.RGBA{R: 40, G: 40, B: 40, A: 255}))
	roadSprite := atlas.Register(render.Solid(color.RGBA{R: 150, G: 130, B: 80, A: 255}))
	redSprite := atlas.Register(render.Solid(color.RGBA{R: 220, G: 90, B: 90, A: 255}))
	blueSprite := atlas.Register(render.Solid(color.RGBA{R: 90, G: 140, B: 220, A: 255}))
	atlas.Close()

	boardCellStyle := func(kind board.CellKind) render.SpriteID {
		switch kind {
		case Wall:
			return wallSprite
		case Road:
			return roadSprite
		default:
			return grassSprite
		}
	}

	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: false},
		Entities: world.EntitiesCfg{MaxCount: MaxEntCount, MinSize: EntitySize, MaxSize: EntitySize},
	}).WithRenderer(atlas)
	boardPlugin := board.NewPlugin(grid, occupancy).WithRenderer(CellSize, atlas, boardCellStyle)
	navigationPlugin := navigation.NewPlugin(UnitSpeed).WithCommands().WithRenderer()
	cameraPlugin := camera.NewPlugin()
	selectionPlugin := selection.NewPlugin().WithRenderer()

	if err := game.UsePlugin(worldPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(boardPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(navigationPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(cameraPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(selectionPlugin); err != nil {
		log.Fatal(err)
	}

	saves, err := game.Persistence.List(saveBasePath)
	if err != nil {
		log.Fatalf("list saves: %v", err)
	}
	hasSave := slices.Contains(saves, "")

	var state *State
	var terrain *board.TerrainMap
	if err := game.Init(func(ctx *gokebiten.GameCtx) error {
		state = &State{}
		ctx.Provide(state)
		terrain = boardPlugin.Terrain()

		if hasSave {
			if err := game.Persistence.Load(saveBasePath, "", state); err != nil {
				log.Fatalf("load: %v", err)
			}
			log.Printf("loaded saved board (save #%d)", state.Saves)
		} else {
			// setup board
			terrain.SetAll(Grass)
			buildWall(grid, terrain)

			// setup entities
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
					start, _ := grid.CellIndex(spawn.startX, spawn.startY)
					return world.Position{AABB: board.CellAABB(grid, start, EntitySize)}
				},
				func(index int) world.Velocity { return world.Velocity{} },
			).
				WithEffect(func(index int) board.Cell {
					spawn := unitRoster[index]
					c, _ := grid.CellIndex(spawn.startX, spawn.startY)
					return board.Cell{ID: c}
				}, func(c board.Cell, id uid.UID64) {
					occupancy.Enter(c.ID, id)
				}).
				With(func(index int) navigation.MoveOrder {
					spawn := unitRoster[index]
					target, _ := grid.CellIndex(spawn.targetX, spawn.startY)
					return navigation.MoveOrder{Target: target}
				}).
				With(func(index int) world.Appearance {
					return world.Appearance{SpriteID: unitRoster[index].sprite}
				}).
				With(func(index int) selection.Selected { return selection.Selected{} })
			worldPlugin.World().Populate(len(unitRoster), spawner)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	game.Loop(func(ctx goke.RunCtx, d time.Duration) {
		navigationPlugin.RunPlan(ctx, d)
		worldPlugin.RunPlan(ctx, d)
		selectionPlugin.RunPlan(ctx, d)
		ctx.Sync()
	})

	game.Layers(boardPlugin.Renderer, worldPlugin.Renderer, navigationPlugin.Renderer, selectionPlugin.Renderer)

	selectionCmdHandler := selectionPlugin.EventHandler()
	navigationCmdHandler := navigationPlugin.EventHandler()
	renderState := game.Resources().Get[*board.RenderState]()
	game.EventHandlerFn(func(events *control.InputEvents) {
		selectionCmdHandler.HandleEvents(events)
		navigationCmdHandler.HandleEvents(events)
		for _, k := range events.KeyEvents {
			if k.Action != control.ActionPress {
				continue
			}
			switch k.Key {
			case ebiten.KeySpace:
				game.TogglePause()
			case ebiten.KeyB:
				renderState.ShowGridLines = !renderState.ShowGridLines
			case ebiten.KeyR:
				buildShortcut(grid, terrain)
				log.Print("built a road through the wall — in-flight units re-path onto it as soon as they deviate")
			case ebiten.KeyF5:
				state.Saves++
				if err := game.Persistence.Save(saveBasePath, "", state); err != nil {
					log.Printf("save: %v", err)
					continue
				}
				log.Printf("saved (save #%d)", state.Saves)
			}
		}
	})
	game.Run()
}

const (
	wallCol     = 12
	shortcutRow = 8
)

func buildWall(grid board.Grid, terrain *board.TerrainMap) {
	var cells []board.CellID
	for y := uint32(2); y < GridHeight; y++ {
		c, _ := grid.CellIndex(wallCol, y)
		cells = append(cells, c)
	}
	terrain.SetMany(cells, Wall)
}

func buildShortcut(grid board.Grid, terrain *board.TerrainMap) {
	c, _ := grid.CellIndex(wallCol, shortcutRow)
	terrain.Set(c, Road)
}
