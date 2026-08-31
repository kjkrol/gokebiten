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
	"github.com/kjkrol/gokebiten/plugins/world/spawners/grid"
	"github.com/kjkrol/gokebiten/plugins/world/spawners/randomvelocity"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/render/atlases/procedural"
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

// State demonstrates persisting arbitrary game-owned state across a save/load cycle.
type State struct{ Saves int }

func main() {
	game := gokebiten.NewGame(&gokebiten.GameProps{
		Title:       "GOKe + GOKg + Ebiten Integration",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	})

	atlas := procedural.NewAtlas()

	worldPlugin := world.NewPlugin(
		world.Config{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		world.Population{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
	).WithRenderer(atlas)

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
		worldPlugin.World().Populate(EntityCount,
			world.NewSpawner(
				grid.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize),
				randomvelocity.New(200, 50, 10),
			),
			world.NewAppearanceExtras(func(index int) world.Appearance {
				return world.Appearance{
					Color:    color.RGBA{R: uint8(rand.IntN(206) + 50), G: uint8(rand.IntN(206) + 50), B: uint8(rand.IntN(206) + 50), A: 255},
					SpriteID: uint8(rand.IntN(procedural.SpriteCount)),
				}
			}),
			collisions.NewCollidableExtras(),
		)
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
				WithOverlay[collisions.Hit](world.Appearance{SpriteID: 0, Color: color.RGBA{R: 255, A: 255}})
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
