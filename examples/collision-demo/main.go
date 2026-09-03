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
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/camera"
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

func main() {
	game := gokebiten.NewGame(&gokebiten.GameProps{
		Title:       "GOKe + GOKg + Ebiten Integration",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	})

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
	var entitySprites [7][4]render.SpriteID
	for ci, c := range palette[:7] {
		for si, shape := range shapes {
			entitySprites[ci][si] = atlas.Register(shape(c))
		}
	}
	hitSprite := atlas.Register(render.Solid(palette[7]))
	atlas.Close()

	worldPlugin := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
	}).WithRenderer(atlas)

	var collisionStats stats.Stats
	collisionsPlugin := collisions.NewPlugin().
		SetCollisionHandlers(elastic.NewHandler(), stats.NewHandler(&collisionStats)).
		SetHitExpires(100 * time.Millisecond)

	cameraPlugin := camera.NewPlugin()

	if err := game.UsePlugin(worldPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(collisionsPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(cameraPlugin); err != nil {
		log.Fatal(err)
	}

	var state *State
	if err := game.Init(func(ctx *gokebiten.GameCtx) error {
		state = &State{}
		ctx.Provide(state)
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	saves, err := game.Persistence.List(saveBasePath)
	if err != nil {
		log.Fatalf("list saves: %v", err)
	}

	if slices.Contains(saves, "") {
		if err := game.Persistence.Load(saveBasePath, "", state); err != nil {
			log.Fatalf("load: %v", err)
		}
		log.Printf("loaded saved world (save #%d)", state.Saves)
	} else {
		placement := world.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize)
		motion := newRandomVelocity(200, 50, 10)
		spawner := world.NewSpawner(
			func(index, count int) world.Position { return placement.Place(index, count) },
			func(index int) world.Velocity { return motion.initialVelocity(index) },
		).
			With(func(index int) world.Appearance {
				return world.Appearance{SpriteID: entitySprites[rand.IntN(7)][rand.IntN(4)]}
			}).
			With(func(index int) collisions.Collision { return collisions.Collision{} })
		worldPlugin.World().Populate(EntityCount, spawner)
	}

	game.Loop(func(ctx goke.RunCtx, d time.Duration) {
		worldPlugin.RunPlan(ctx, d)
		collisionsPlugin.RunPlan(ctx, d)
		ctx.Sync()
	})

	game.Layers(
		func() render.Renderer {
			return render.NewCachedRenderer(
				render.SolidBackground{Color: color.RGBA{R: 50, G: 50, B: 50, A: 255}},
				ScreenWidth, ScreenHeight,
			)
		},
		func() render.Renderer {
			return worldPlugin.EntityRenderer().
				WithOverlay[collisions.Hit](world.Appearance{SpriteID: hitSprite})
		},
		func() render.Renderer {
			kin := game.Resources().Get[*world.Telemetry]()
			entityCount := func() int { return kin.Count }
			return render.NewTelemetryRenderer(&game.Resources().Get[*gokebiten.TPS]().Ticks, entityCount, &collisionStats.Counter)
		},
	)

	game.EventHandlerFn(func(events *control.InputEvents) {
		for _, k := range events.KeyEvents {
			if k.Action != control.ActionPress {
				continue
			}
			switch k.Key {
			case ebiten.KeySpace:
				game.TogglePause()
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
