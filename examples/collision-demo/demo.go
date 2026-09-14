package main

import (
	"fmt"
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

// =========================== Game ===========================

// Demo is the collision demo — exactly one Stage (mainStage below).
type Demo struct{ stage *mainStage }

var _ game.Game = (*Demo)(nil)

func NewDemo() *Demo { return &Demo{stage: &mainStage{}} }

func (d *Demo) Props() game.Props {
	return game.Props{
		Title:       "GOKe + GOKg + Ebiten Integration",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	}
}

func (d *Demo) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{d.stage.Name(): d.stage}, d.stage.Name()
}

// =========================== Stage ===========================

// State  persisting arbitrary game-owned state across a save/load cycle.
type State struct{ Saves int }

const (
	entityColors = 7
	entityShapes = 4
	hitKind      = "hit"
)

// body is a roster entry's Data for every entity kind: its starting position and velocity.
type body struct {
	pos world.Position
	vel world.Velocity
}

// entityKindName names the EntKind drawn with color ci and shape si; hitKind only supplies the overlay sprite.
func entityKindName(ci, si int) string { return fmt.Sprintf("entity-%d-%d", ci, si) }

type mainStage struct {
	world      *world.Plugin
	collisions *collisions.Plugin

	state          *State
	collisionStats stats.Stats

	stack game.Scenes
}

var _ game.Stage = (*mainStage)(nil)

func (s *mainStage) Name() string { return "collision-demo" }

func (s *mainStage) Stack() game.Scenes { return s.stack }

func (s *mainStage) Init(ctx game.Initializer) error {
	s.world = ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: EntityCount, MinSize: RectSize, MaxSize: RectSize},
	})
	kinds := s.world.EntKindDict()
	for ci := range entityColors {
		for si := range entityShapes {
			kinds.Define(entityKindName(ci, si), func(k world.Kind[body]) world.EntKind {
				return world.EntKind{
					Position:   k.Load(func(b body) world.Position { return b.pos }),
					Velocity:   k.Load(func(b body) world.Velocity { return b.vel }),
					Components: []world.ComponentTemplate{k.Const(collisions.Collision{})},
				}
			})
		}
	}
	kinds.Define(hitKind, func(world.Kind[struct{}]) world.EntKind { return world.EntKind{} })

	s.collisions = collisions.NewPlugin(100*time.Millisecond, s.world).
		SetCollisionHandlers(elastic.NewHandler(), stats.NewHandler(&s.collisionStats))
	s.state = &State{}
	if err := ctx.Use(s.collisions); err != nil {
		return err
	}

	main := &mainScene{stage: s, tps: ctx.TPS()}
	stack, err := game.NewStack(main)
	if err != nil {
		return err
	}
	s.stack = stack
	comp := stack.Composition()
	comp.Show(main.Name())
	return ctx.Track(comp)
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
	log.Printf("loaded saved world (save #%d)", s.state.Saves)
	return true, nil
}

func (s *mainStage) Spawn() error {
	placement := world.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize)
	motion := newRandomVelocity(200, 50, 10)
	kinds := s.world.EntKindDict()
	entries := make([]world.Entry, EntityCount)
	for i := range entries {
		entries[i] = kinds.Entry(entityKindName(rand.IntN(entityColors), rand.IntN(entityShapes)),
			body{pos: placement.Place(i, EntityCount), vel: motion.initialVelocity(i)})
	}
	s.world.Seed(entries...)
	return nil
}

func (s *mainStage) Update(ctx goke.RunCtx, d time.Duration) {
	s.world.RunPlan(ctx, d)
	s.collisions.RunPlan(ctx, d)
	ctx.Sync()
}

// =========================== Scene ===========================

type mainScene struct {
	stage *mainStage
	tps   *game.TPS
}

var _ game.Scene = (*mainScene)(nil)

func (m *mainScene) Name() string { return "main" }

func (m *mainScene) Layers() []func() render.Renderer {
	s := m.stage

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
	kinds := s.world.EntKindDict()
	atlas := render.NewAtlas(RectSize, len(kinds.All()))
	shapes := [entityShapes]func(color.RGBA) render.SpriteDrawer{render.Solid, render.Border, render.Diamond, render.Cross}
	for ci, c := range palette[:entityColors] {
		for si, shape := range shapes {
			kind, _ := kinds.Get(entityKindName(ci, si))
			atlas.RegisterAt(kind.SpriteID, shape(c))
		}
	}
	hit, _ := kinds.Get(hitKind)
	atlas.RegisterAt(hit.SpriteID, render.Solid(palette[entityColors]))
	atlas.Close()
	s.world.WithRenderer(atlas)

	return []func() render.Renderer{
		func() render.Renderer {
			return render.NewCachedRenderer(
				render.SolidBackground{Color: color.RGBA{R: 50, G: 50, B: 50, A: 255}},
				ScreenWidth, ScreenHeight,
			)
		},
		func() render.Renderer {
			return s.world.EntityRenderer().
				WithOverlay[collisions.Hit](world.Appearance{SpriteID: hit.SpriteID})
		},
		func() render.Renderer {
			kin := s.world.Res.Telemetry
			entityCount := func() int { return kin.Count }
			return render.NewTelemetryRenderer(&m.tps.Ticks, entityCount, &s.collisionStats.Counter)
		},
	}
}

func (m *mainScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, composition game.Composition) {
	s := m.stage
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
