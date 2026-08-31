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
	"github.com/kjkrol/gokebiten/plugins/selection"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/render/atlases/procedural"
	"github.com/kjkrol/gokg/geom"
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

	wallCol      = 12
	shortcutRow  = 8
	saveBasePath = "board-rts-demo"
)

// State demonstrates persisting arbitrary game-owned state across a
// save/load cycle, alongside the mutable terrain — see main's Persistence calls.
type State struct{ Saves int }

func main() {
	game := gokebiten.NewGame(&gokebiten.GameProps{
		Title:       "gokebiten board plugin — square grid RTS demo",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	})

	grid := board.NewSquareGrid(GridWidth, GridHeight, CellSize)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	buildWall(grid, terrain)
	occupancy := &board.SingleOccupancy{}

	atlas := procedural.NewAtlas()

	worldPlugin := world.NewPlugin(
		world.Config{Width: ScreenWidth, Height: ScreenHeight, Toroidal: false},
		world.Population{MaxCount: len(spawns), MinSize: EntitySize, MaxSize: EntitySize},
	).WithRenderer(atlas)
	boardPlugin := board.NewPlugin(grid, terrain, occupancy, UnitSpeed).WithRenderer(CellSize, boardCellStyle).WithCommands()
	cameraPlugin := camera.NewPlugin()
	selectionPlugin := selection.NewPlugin()

	if err := game.UsePlugin(worldPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(boardPlugin); err != nil {
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
	var cellWatcher goke.Runnable
	var autoSelectHandle goke.Runnable
	if err := game.Init(func(ctx *gokebiten.GameCtx) error {
		state = &State{}
		ctx.Provide(state)
		cellWatcher = ctx.RegSys(func() goke.System { return newCellEnteredWatcher() })
		if !hasSave {
			spawner := newUnitSpawner(grid, occupancy)
			worldPlugin.World().Populate(len(spawns), spawner)
			autoSelectHandle = ctx.RegSys(func() goke.System {
				return &autoSelector{spawner: spawner, sys: selectionPlugin.System()}
			})
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	if hasSave {
		if err := game.Persistence.Load(saveBasePath, "", state, terrain); err != nil {
			log.Fatalf("load: %v", err)
		}
		log.Printf("loaded saved board (save #%d)", state.Saves)
	}

	pathRenderer := board.NewPathRenderer(boardPlugin.Board())
	selectionRenderer := selection.NewRenderer(selectionPlugin.State())

	game.Loop(func(ctx goke.RunCtx, d time.Duration) {
		if autoSelectHandle != nil {
			ctx.Run(autoSelectHandle, d)
		}
		boardPlugin.RunPlan(ctx, d)
		worldPlugin.RunPlan(ctx, d)
		selectionPlugin.RunPlan(ctx, d)
		ctx.Run(cellWatcher, d)
		ctx.Sync()
	})

	game.Layers(boardPlugin.Renderer, worldPlugin.Renderer,
		func() render.Renderer { return pathRenderer },
		func() render.Renderer { return selectionRenderer })

	selHandler := selectionPlugin.EventHandler()
	cmdHandler := boardPlugin.EventHandler()
	game.EventHandlerFn(func(events *control.InputEvents) {
		selHandler.HandleEvents(events)
		cmdHandler.HandleEvents(events)
		for _, k := range events.KeyEvents {
			if k.Action != control.ActionPress {
				continue
			}
			switch k.Key {
			case ebiten.KeySpace:
				game.TogglePause()
			case ebiten.KeyB:
				boardPlugin.CellRenderer().SetShowGridLines(!boardShowsGrid)
				boardShowsGrid = !boardShowsGrid
			case ebiten.KeyR:
				buildShortcut(grid, terrain)
				log.Print("built a road through the wall — in-flight units re-path onto it as soon as they deviate")
			case ebiten.KeyF5:
				state.Saves++
				if err := game.Persistence.Save(saveBasePath, "", state, terrain); err != nil {
					log.Printf("save: %v", err)
					continue
				}
				log.Printf("saved (save #%d)", state.Saves)
			}
		}
	})
	game.Run()
}

// boardShowsGrid mirrors Renderer's internal toggle state so the B key
// can flip it — Renderer has no getter since nothing else needs to read it back.
var boardShowsGrid = true

// buildWall makes column wallCol impassable except its top two rows, so a
// unit on the left must detour to the top to reach the right side.
func buildWall(grid *board.SquareGrid, terrain *board.TerrainMap) {
	for y := uint32(2); y < GridHeight; y++ {
		terrain.Set(cellAt(grid, wallCol, y), board.CellProps{Cost: 1, Passable: false})
	}
}

// buildShortcut opens a fast gap through the wall at shortcutRow — pressing
// R calls this live, demonstrating that SteeringSystem re-paths around
// changed terrain as soon as an in-flight unit next deviates from its route.
func buildShortcut(grid *board.SquareGrid, terrain *board.TerrainMap) {
	terrain.Set(cellAt(grid, wallCol, shortcutRow), board.CellProps{Cost: 0.4, Passable: true})
}

func cellAt(grid *board.SquareGrid, x, y uint32) board.CellID {
	c, _ := grid.CellAt(geom.NewVec(float64(x)*CellSize+1, float64(y)*CellSize+1))
	return c
}

var (
	colorImpassable = color.RGBA{R: 40, G: 40, B: 40, A: 255}
	colorRoad       = color.RGBA{R: 150, G: 130, B: 80, A: 255}
	colorGround     = color.RGBA{R: 60, G: 95, B: 60, A: 255}
)

// boardCellStyle is this demo's board.CellStyle: dark for walls, a lighter
// tan for anything cheaper than the baseline (roads), green otherwise.
func boardCellStyle(_ board.CellID, cost float64, passable bool) color.RGBA {
	switch {
	case !passable:
		return colorImpassable
	case cost < 1:
		return colorRoad
	default:
		return colorGround
	}
}

var spawns = []struct {
	startX, startY uint32
	targetX        uint32
	color          color.RGBA
}{
	{startX: 2, startY: 4, targetX: GridWidth - 3, color: color.RGBA{R: 220, G: 90, B: 90, A: 255}},
	{startX: 2, startY: 12, targetX: GridWidth - 3, color: color.RGBA{R: 90, G: 140, B: 220, A: 255}},
}

// unitSpawner places each unit at spawns[index]'s start cell and enters it
// into occupancy — implements world.Spawner (populators[0] for Populate).
type unitSpawner struct {
	grid      *board.SquareGrid
	occupancy *board.SingleOccupancy
	lastStart board.CellID
	spawned   []uid.UID64

	cell   goke.Comp[board.Cell]
	moveTo goke.Comp[board.MoveTo]
	path   goke.Comp[board.Path]
	app    goke.Comp[world.Appearance]
}

func newUnitSpawner(grid *board.SquareGrid, occupancy *board.SingleOccupancy) *unitSpawner {
	return &unitSpawner{grid: grid, occupancy: occupancy}
}

func (u *unitSpawner) Spawn(index, count int) (world.Position, world.Velocity) {
	spawn := spawns[index]
	u.lastStart = cellAt(u.grid, spawn.startX, spawn.startY)
	return world.Position{AABB: board.CellAABB(u.grid, u.lastStart, EntitySize)}, world.Velocity{}
}

func (u *unitSpawner) Components() []goke.Addable {
	return []goke.Addable{&u.cell, &u.moveTo, &u.path, &u.app}
}

func (u *unitSpawner) Init(cursor *goke.Cursor, i, index int, id uid.UID64) {
	spawn := spawns[index]
	target := cellAt(u.grid, spawn.targetX, spawn.startY)

	u.cell.Slice(cursor)[i] = board.Cell{ID: u.lastStart}
	u.moveTo.Slice(cursor)[i] = board.MoveTo{Target: target}
	u.app.Slice(cursor)[i] = world.Appearance{Color: spawn.color, SpriteID: 0}

	u.occupancy.Enter(u.lastStart, id)
	u.spawned = append(u.spawned, id)
}

// autoSelector tags every unit unitSpawner spawned as Selected, once, on
// its first tick — Populate only queues the spawn, so the entity IDs aren't
// real until the ECS's one-time Setup flush has run.
type autoSelector struct {
	spawner *unitSpawner
	sys     *selection.System
	done    bool
}

func (a *autoSelector) Init(*goke.SysInit) {}

func (a *autoSelector) Update(_ *goke.CmdBuf, _ time.Duration) {
	if a.done {
		return
	}
	a.done = true
	a.sys.Select(a.spawner.spawned)
}

// cellEnteredWatcher demonstrates reacting to board.CellEntered from
// outside the board package — it never touches board's internals, only the
// public tag component.
type cellEnteredWatcher struct {
	query *goke.Query
	tag   goke.Comp[board.CellEntered]
}

func newCellEnteredWatcher() *cellEnteredWatcher { return &cellEnteredWatcher{} }

func (w *cellEnteredWatcher) Init(si *goke.SysInit) {
	w.query = si.NewQueryBuilder(&w.tag).Build()
}

func (w *cellEnteredWatcher) Update(_ *goke.CmdBuf, _ time.Duration) {
	w.query.All()
	for w.query.Next() {
		cursor := w.query.Cursor()
		tags := w.tag.Slice(cursor)
		for i, id := range cursor.IDs {
			log.Printf("unit %v entered cell %v", id, tags[i].ID)
		}
	}
}
