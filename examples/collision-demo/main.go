package main

import (
	"encoding/binary"
	"fmt"
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
	"github.com/kjkrol/gokebiten/physics"
	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/elastic"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/stats"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/grid"
	"github.com/kjkrol/gokebiten/physics/kinematics/spawners/randomvelocity"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/render/atlases/procedural"
	"github.com/kjkrol/gokebiten/spatial"
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

// State demonstrates persisting arbitrary game-owned state alongside the
// ECS snapshot — Saves is otherwise meaningless, just something to observe
// round-tripping across a save/load cycle.
type State struct{ Saves int }

func main() {
	resources := gokebiten.NewResources(
		&gokebiten.GameProps{
			Title:       "GOKe + GOKg + Ebiten Integration",
			ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
			TargetTPS: TPS},
		spatial.Config{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		State{},
		Telemetry{},
	)

	game := gokebiten.NewGame(resources)
	game.SetEventHandler(&EventHandler{game: game, resources: resources})

	worldModule := spatial.NewWorldModule(
		resources.GetSpaceConfig(),
		spatial.Population{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
	)

	physicsModule := physics.New(worldModule.Space(), game.ECS(), RectSize, game.Step())
	physicsModule.SetCollisionHandlers(
		elastic.NewHandler(),
		stats.NewHandler(&resources.Telemetry().Collision),
	)
	physicsModule.SetHitExpires(100 * time.Millisecond)

	saves, err := gokebiten.ListSaves(saveBasePath)
	if err != nil {
		log.Fatalf("list saves: %v", err)
	}
	if slices.Contains(saves, "") {
		if err := loadExistingWorld(game, worldModule, resources, physicsModule); err != nil {
			log.Fatalf("load: %v", err)
		}
		log.Printf("loaded saved world (save #%d)", resources.State().Saves)
	} else {
		populateFreshWorld(worldModule, &resources.Telemetry().Kinematics)
	}

	game.Setup(worldModule)
	game.UseModule(physicsModule)

	game.Loop(func(ctx goke.RunCtx, d time.Duration) {
		physicsModule.RunPlan(ctx, d)
		ctx.Sync()
	})

	camera := render.NewCamera(ScreenWidth, ScreenHeight)
	atlas := procedural.NewAtlas()

	game.RenderSequence(
		func() render.Renderer {
			return render.NewEntitiesRenderer(atlas, camera, goke.Exclude[collisions.Hit]())
		},
		func() render.Renderer {
			return render.NewTagOverlayRenderer(atlas, camera, 0, color.RGBA{R: 255, A: 255}, goke.Include[collisions.Hit]())
		},
		func() render.Renderer {
			kin := &resources.Telemetry().Kinematics
			entityCount := func() int { return kin.DynamicCount + kin.StaticCount }
			return render.NewTelemetryRenderer(resources.TPS(), entityCount, &resources.Telemetry().Collision.Counter)
		},
	)

	game.Run()
}

func (s State) MarshalBinary() ([]byte, error) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(s.Saves))
	return b, nil
}

func (s *State) UnmarshalBinary(data []byte) error {
	if len(data) != 4 {
		return fmt.Errorf("state: want 4 bytes, got %d", len(data))
	}
	s.Saves = int(binary.BigEndian.Uint32(data))
	return nil
}

type Telemetry struct {
	Kinematics kinematics.Telemetry
	Collision  stats.Stats
}

func (t *Telemetry) Reset() { t.Collision.Counter = 0 }

var _ control.EventHandler = (*EventHandler)(nil)

type EventHandler struct {
	game      *gokebiten.Game
	resources *gokebiten.Resources[State, Telemetry]
}

func (e *EventHandler) HandleEvents(events *control.InputEvents) {
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeySpace:
			e.game.TogglePause()
		case ebiten.KeyF5:
			e.save()
		}
	}
}

func (e *EventHandler) save() {
	e.resources.State().Saves++
	if err := e.game.Save(saveBasePath, ""); err != nil {
		log.Printf("save: %v", err)
		return
	}
	log.Printf("saved (save #%d)", e.resources.State().Saves)
}

func loadExistingWorld(game *gokebiten.Game, worldModule *spatial.WorldModule, resources *gokebiten.Resources[State, Telemetry], physicsModule *physics.Physics) error {
	onLoaded := func(count int) { resources.Telemetry().Kinematics.DynamicCount = count }
	return game.Load(saveBasePath, "", worldModule.Space(), onLoaded, physicsModule)
}

func populateFreshWorld(worldModule *spatial.WorldModule, telemetry *kinematics.Telemetry) {
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
