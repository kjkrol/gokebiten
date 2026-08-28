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
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/physics"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/elastic"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/grid"
	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/randomvelocity"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/render/atlases/procedural"
	"github.com/kjkrol/gokebiten/world"
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

	worldPlugin := world.NewPlugin(
		world.Config{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		world.Population{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
	)

	physicsPlugin := physics.NewPlugin(RectSize).
		SetCollisionHandlers(elastic.NewHandler()).
		SetHitExpires(100 * time.Millisecond).
		EnableStats()

	cameraPlugin := camera.NewPlugin()

	if err := game.UsePlugin(worldPlugin); err != nil {
		log.Fatal(err)
	}
	if err := game.UsePlugin(physicsPlugin); err != nil {
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
		populateFreshWorld(worldPlugin.World())
	}

	game.Loop(func(ctx goke.RunCtx, d time.Duration) {
		physicsPlugin.Physics().RunPlan(ctx, d)
		ctx.Sync()
	})

	atlas := procedural.NewAtlas()

	game.Layers(
		func() render.Layer {
			return render.NewCachedLayer(
				render.SolidBackground{Color: color.RGBA{R: 50, G: 50, B: 50, A: 255}},
				ScreenWidth, ScreenHeight,
			)
		},
		func() render.Layer {
			return render.NewEntitiesRenderer(atlas, cameraPlugin.Camera()).
				WithOverlay[collisions.Hit](render.Appearance{SpriteID: 0, Color: color.RGBA{R: 255, A: 255}})
		},
		func() render.Layer {
			kin := game.Resources().Get[*world.Telemetry]()
			entityCount := func() int { return kin.DynamicCount + kin.StaticCount }
			return render.NewTelemetryRenderer(&game.Resources().Get[*gokebiten.TPS]().Ticks, entityCount, &physicsPlugin.Stats().Counter)
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

func populateFreshWorld(worldModule *world.Module) {
	worldModule.Populate(EntityCount,
		kinematics.NewSpawner(
			grid.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize),
			randomvelocity.New(200, 50, 10),
		),
		render.NewAppearanceExtras(func(index int) render.Appearance {
			return render.Appearance{
				Color:    color.RGBA{R: uint8(rand.IntN(206) + 50), G: uint8(rand.IntN(206) + 50), B: uint8(rand.IntN(206) + 50), A: 255},
				SpriteID: uint8(rand.IntN(procedural.SpriteCount)),
			}
		}),
	)
}
