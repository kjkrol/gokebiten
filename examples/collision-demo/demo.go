package main

import (
	"image/color"
	"log"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/elastic"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/stats"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

const (
	TPS          = 60 * 2
	ScreenWidth  = 1024
	ScreenHeight = 1024
	RectSize     = 20
	FillPercent  = 20

	saveBasePath = "collision-demo"
)

var EntityCount = int(math.Floor(FillPercent / 100.0 * float64(ScreenWidth*ScreenHeight) / float64(RectSize*RectSize)))

// State  persisting arbitrary game-owned state across a save/load cycle.
type State struct{ Saves int }

// Demo wires the collision demo — its plugins are its own fields, built in Init.
type Demo struct {
	world      *world.Plugin
	collisions *collisions.Plugin

	state          *State
	collisionStats stats.Stats
	hitSprite      render.SpriteID
	entitySprites  [7][4]render.SpriteID
}

var _ game.Game = (*Demo)(nil)

func (dm *Demo) Init(ctx game.Initializer) error {
	palette := [8]color.RGBA{
		{R: 80, G: 120, B: 220, A: 255},  // blue
		{R: 90, G: 200, B: 110, A: 255},  // green
		{R: 80, G: 200, B: 210, A: 255},  // cyan
		{R: 150, G: 100, B: 220, A: 255}, // purple
		{R: 220, G: 210, B: 80, A: 255},  // yellow
		{R: 230, G: 160, B: 60, A: 255},  // amber
		{R: 60, G: 160, B: 150, A: 255},  // teal
		{R: 220, G: 40, B: 40, A: 255},   // red — reserved for the hit sprite, not an entity color
	}
	atlas := render.NewAtlas(RectSize, 28*4+1)
	shapes := [4]func(color.RGBA) render.SpriteDrawer{render.Solid, render.Border, render.Diamond, render.Cross}
	for ci, c := range palette[:7] {
		for si, shape := range shapes {
			dm.entitySprites[ci][si] = atlas.Register(shape(c))
		}
	}
	dm.hitSprite = atlas.Register(render.Solid(palette[7]))
	atlas.Close()

	dm.world = ctx.World()
	dm.world.WithRenderer(atlas)
	dm.world.WithCameraControls()

	dm.collisions = collisions.NewPlugin(100*time.Millisecond, dm.world).
		SetCollisionHandlers(elastic.NewHandler(), stats.NewHandler(&dm.collisionStats))
	dm.state = &State{}
	return ctx.Use(dm.collisions)
}

func (dm *Demo) Restore(p game.Persistence) (bool, error) {
	saves, err := p.List(saveBasePath)
	if err != nil {
		return false, err
	}
	if !slices.Contains(saves, "") {
		return false, nil
	}
	if err := p.Load(saveBasePath, "", dm.state); err != nil {
		return false, err
	}
	log.Printf("loaded saved world (save #%d)", dm.state.Saves)
	return true, nil
}

func (dm *Demo) Spawn() ([]world.Batch, error) {
	placement := world.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize)
	motion := newRandomVelocity(200, 50, 10)
	spawner := world.NewSpawner(
		func(index, count int) world.Position { return placement.Place(index, count) },
		func(index int) world.Velocity { return motion.initialVelocity(index) },
	).
		With(func(index int) world.Appearance {
			return world.Appearance{SpriteID: dm.entitySprites[rand.IntN(7)][rand.IntN(4)]}
		}).
		With(func(index int) collisions.Collision { return collisions.Collision{} })
	return []world.Batch{{Count: EntityCount, Spawner: spawner}}, nil
}

func (dm *Demo) Update(ctx goke.RunCtx, d time.Duration) {
	dm.world.RunPlan(ctx, d)
	dm.collisions.RunPlan(ctx, d)
	ctx.Sync()
}

func (dm *Demo) Draw(runtime game.Runtime) []func() render.Renderer {
	return []func() render.Renderer{
		func() render.Renderer {
			return render.NewCachedRenderer(
				render.SolidBackground{Color: color.RGBA{R: 50, G: 50, B: 50, A: 255}},
				ScreenWidth, ScreenHeight,
			)
		},
		func() render.Renderer {
			return dm.world.EntityRenderer().
				WithOverlay[collisions.Hit](world.Appearance{SpriteID: dm.hitSprite})
		},
		func() render.Renderer {
			kin := dm.world.Res.Telemetry
			entityCount := func() int { return kin.Count }
			return render.NewTelemetryRenderer(&runtime.TPS().Ticks, entityCount, &dm.collisionStats.Counter)
		},
	}
}

func (dm *Demo) HandleEvents(events *control.InputEvents, runtime game.Runtime) {
	dm.world.EventHandler().HandleEvents(events)
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeyEscape:
			runtime.Quit()
		case ebiten.KeySpace:
			runtime.TogglePause()
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
