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
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/stats"
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

	state := &State{}
	telemetry := &Telemetry{}
	game.Resources().InsertResource(state)
	game.Resources().InsertResource(telemetry)

	game.SetEventHandlerFn(func(events *control.InputEvents) {
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

	worldPlugin := world.NewPlugin(
		world.Config{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		world.Population{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
	).OnReindexed(func(count int) { telemetry.Kinematics.DynamicCount = count })

	physicsPlugin := physics.NewPlugin(RectSize).
		SetCollisionHandlers(elastic.NewHandler(), stats.NewHandler(&telemetry.Collision)).
		SetHitExpires(100 * time.Millisecond)

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
		populateFreshWorld(worldPlugin.World(), &telemetry.Kinematics)
	}

	game.Loop(func(ctx goke.RunCtx, d time.Duration) {
		physicsPlugin.Physics().RunPlan(ctx, d)
		ctx.Sync()
	})

	atlas := procedural.NewAtlas()

	game.RenderSequence(
		func() render.Renderer {
			return render.NewEntitiesRenderer(atlas, cameraPlugin.Camera(), goke.Exclude[collisions.Hit]())
		},
		func() render.Renderer {
			return render.NewTagOverlayRenderer(atlas, cameraPlugin.Camera(), 0, color.RGBA{R: 255, A: 255}, goke.Include[collisions.Hit]())
		},
		func() render.Renderer {
			kin := &telemetry.Kinematics
			entityCount := func() int { return kin.DynamicCount + kin.StaticCount }
			return render.NewTelemetryRenderer(&game.Resources().GetResource[*gokebiten.TPS]().Ticks, entityCount, &telemetry.Collision.Counter)
		},
	)

	game.Run()
}

type Telemetry struct {
	Kinematics kinematics.Telemetry
	Collision  stats.Stats
}

func (t *Telemetry) Reset() { t.Collision.Counter = 0 }

func populateFreshWorld(worldModule *world.Module, telemetry *kinematics.Telemetry) {
	worldModule.Populate(EntityCount, telemetry,
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
