// Command vision-demo shows ten entities keeping out of each other's way by
// sight, and one red hunter that lives off the ones who fail at it.
//
// Everything here carries a Sight cone, drawn on screen. The prey give way to
// whatever their cone says is on a collision course — something bearing down on
// them, or their own heading pointing into someone — and pass by everything
// else — bar the hunter, which is broken away from the moment it comes into
// view, whichever way it happens to be drifting. It goes the other way itself:
// it steers at the nearest prey it sees — or, seeing none, looks a quarter turn
// to one side, runs on, and looks again — and whatever it touches is gone, which
// is a world.Despawn from an ordinary behavior reading the contact. It is a
// tenth slower than what it chases, so it only ever catches bad steering.
//
// Press A to switch the avoidance off and watch the entity count fall.
//
// The world wraps, so a cone reaching past an edge is drawn again on the far
// side — and an entity sees through the seam just as it moves through it.
package main

import (
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/stats"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/vision/strategies/flee"
	"github.com/kjkrol/gokebiten/plugins/vision/strategies/hunt"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
)

const (
	TPS          = 60
	ScreenWidth  = 1024
	ScreenHeight = 768

	PreyCount = 10
	RectSize  = 16

	sightRadius = 200
	sightHalf   = math.Pi / 5
	roamSpeed   = 90
	// The hunter is the slower one: it only ever catches what steers badly.
	hunterSpeed = roamSpeed * 0.9
	// With nobody in view the hunter looks to one side, runs on a second, and
	// looks again — long enough to cover ground, short enough to keep scanning.
	hunterLooksEvery = time.Second
	preyKind         = "prey"
	hunterKind       = "hunter"
	backdropGrey     = 40
)

// =========================== Game ===========================

// Demo is the vision demo — exactly one Stage.
type Demo struct{ stage *mainStage }

var _ game.Game = (*Demo)(nil)

func NewDemo() *Demo { return &Demo{stage: &mainStage{avoiding: true}} }

func (d *Demo) Props() game.Props {
	return game.Props{
		Title:       "gokebiten — sight, avoidance and a hunter",
		ScreenWidth: ScreenWidth, ScreenHeight: ScreenHeight,
		TargetTPS: TPS,
	}
}

func (d *Demo) Stages() (map[string]game.Stage, string) {
	return map[string]game.Stage{d.stage.Name(): d.stage}, d.stage.Name()
}

// =========================== Stage ===========================

// body is a roster entry's data: where an entity starts and where it heads.
type body struct {
	pos world.Position
	vel world.Velocity
}

type mainStage struct {
	world      *world.Plugin
	vision     *vision.Plugin
	collisions *collisions.Plugin

	avoidance *flee.Behavior
	avoiding  bool
	hits      stats.Stats

	stack game.Scenes
}

var _ game.Stage = (*mainStage)(nil)

func (s *mainStage) Name() string       { return "vision-demo" }
func (s *mainStage) Stack() game.Scenes { return s.stack }

func (s *mainStage) Init(ctx game.Initializer) error {
	s.world = ctx.UseWorld(world.Config{
		Space:    world.SpaceCfg{Width: ScreenWidth, Height: ScreenHeight, Toroidal: true},
		Entities: world.EntitiesCfg{MaxCount: PreyCount + 1, MinSize: RectSize, MaxSize: RectSize},
	})

	kinds := s.world.EntKindDict()
	kinds.Define(preyKind, func(k world.Kind[body]) world.EntKind {
		return world.EntKind{
			Position: k.Load(func(b body) world.Position { return b.pos }),
			Velocity: k.Load(func(b body) world.Velocity { return b.vel }),
			Components: append(s.sees(k),
				k.Const(world.Steering{Reflex: 3, TurnRate: 0.12}),
				k.Const(flee.Skittish{}),
				k.Const(hunt.Prey{}),
				k.Const(collisions.Physics{Restitution: 1})),
		}
	})
	kinds.Define(hunterKind, func(k world.Kind[body]) world.EntKind {
		return world.EntKind{
			Position: k.Load(func(b body) world.Position { return b.pos }),
			Velocity: k.Load(func(b body) world.Velocity { return b.vel }),
			Components: append(s.sees(k),
				k.Const(world.Steering{Reflex: 1, TurnRate: 0.30}),
				k.Const(hunt.Predator{}), k.Const(flee.Threat{})),
		}
	})

	s.avoidance = flee.New()

	s.vision = vision.NewPlugin(s.world)
	if err := s.vision.RegisterBehavior(
		plugin.Between[flee.Skittish, plugin.Anything](s.avoidance.Steer, plugin.Asking[flee.Threat]()),
		plugin.Between[hunt.Predator, hunt.Prey](hunt.Chase(hunterLooksEvery)),
		plugin.Between[plugin.Anything, plugin.Anything](faceTravel),
	); err != nil {
		return err
	}
	s.collisions = collisions.NewPlugin(s.world)
	if err := s.collisions.RegisterBehavior(
		plugin.Between[plugin.Anything, plugin.Anything](stats.Count(&s.hits)),
		plugin.Between[hunt.Predator, hunt.Prey](s.caught),
	); err != nil {
		return err
	}

	if err := ctx.Use(s.vision); err != nil {
		return err
	}
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

// sees is what every kind in this demo shares: a cone that looks where the
// entity is going, a steady hand on the tiller, and a body that collides.
func (s *mainStage) sees(k world.Kind[body]) []world.ComponentTemplate {
	return []world.ComponentTemplate{
		// Facing is set from the entity's own heading at spawn; the scan keeps
		// looking wherever the entity was last pointed.
		k.Load(func(b body) vision.Sight {
			return vision.Sight{Facing: b.vel.Dir, HalfAngle: sightHalf, Radius: sightRadius}
		}),
		k.Const(vision.SightOutline{}),
		collisions.Collidable(s.world.Space()),
	}
}

func (s *mainStage) Restore(game.Persistence) (bool, error) { return false, nil }

func (s *mainStage) Spawn() error {
	kinds := s.world.EntKindDict()
	placement := world.NewGridPlacement(ScreenWidth, ScreenHeight, RectSize)

	const total = PreyCount + 1
	roam := func(i int, speed float64) body {
		a := rand.Float64() * 2 * math.Pi
		return body{
			pos: placement.Place(i, total),
			vel: world.Velocity{Dir: geom.NewVec(math.Cos(a), math.Sin(a)), Value: speed},
		}
	}

	entries := make([]world.Entry, 0, total)
	for i := range PreyCount {
		entries = append(entries, kinds.Entry(preyKind, roam(i, roamSpeed)))
	}
	entries = append(entries, kinds.Entry(hunterKind, roam(PreyCount, hunterSpeed)))
	s.world.Seed(entries...)
	return nil
}

func (s *mainStage) Update(ctx goke.RunCtx, d time.Duration) {
	// vision first: this tick's behaviors read what the last scan found.
	s.vision.RunPlan(ctx, d)
	s.world.RunPlan(ctx, d)
	s.collisions.RunPlan(ctx, d)
	ctx.Sync()
}

// caught is what a catch means in this demo: the hunter touches a prey, and
// the prey is gone.
func (s *mainStage) caught(t plugin.Tick, m collisions.Meeting) {
	s.world.Despawn(t.Cmd, m.Other)
}

// faceTravel points each entity's Sight where it is actually going, so the cone
// follows the swerve instead of staring at where the entity set off.
func faceTravel(_ plugin.Tick, s vision.Sighting) {
	if d := s.Base.Vel.Dir; d.X != 0 || d.Y != 0 {
		s.Sight.Facing = d
	}
}

// =========================== Scene ===========================

type mainScene struct {
	stage *mainStage
	tps   *game.TPS
}

var _ game.Scene = (*mainScene)(nil)

func (m *mainScene) Name() string    { return "main" }
func (m *mainScene) Focusable() bool { return true }

func (m *mainScene) Layers() []func() render.Renderer {
	s := m.stage

	kinds := s.world.EntKindDict()
	atlas := render.NewAtlas(RectSize, len(kinds.All()))
	prey, _ := kinds.Get(preyKind)
	hunter, _ := kinds.Get(hunterKind)
	atlas.RegisterAt(prey.SpriteID, render.Solid(color.RGBA{R: 120, G: 190, B: 255, A: 255}))
	atlas.RegisterAt(hunter.SpriteID, render.Solid(color.RGBA{R: 225, G: 70, B: 70, A: 255}))
	atlas.Close()
	s.world.WithRenderer(atlas)
	s.vision.WithRenderer(atlas)

	return []func() render.Renderer{
		func() render.Renderer {
			return render.NewCachedRenderer(
				render.SolidBackground{Color: color.RGBA{R: backdropGrey, G: backdropGrey, B: backdropGrey + 6, A: 255}},
				ScreenWidth, ScreenHeight,
			)
		},
		func() render.Renderer { return s.vision.Renderer() },
		func() render.Renderer { return s.world.EntityRenderer() },
		func() render.Renderer {
			count := func() int { return s.world.Res.Telemetry.Count }
			return render.NewTelemetryRenderer(&m.tps.Ticks, count, &s.hits.Counter)
		},
	}
}

func (m *mainScene) HandleEvents(events *control.InputEvents, runtime game.Runtime, _ game.Composition) {
	for _, k := range events.KeyEvents {
		if k.Action != control.ActionPress {
			continue
		}
		switch k.Key {
		case ebiten.KeyEscape:
			runtime.Quit()
		case ebiten.KeySpace:
			runtime.TogglePause()
		case ebiten.KeyA:
			m.stage.avoiding = !m.stage.avoiding
			m.stage.avoidance.SetEnabled(m.stage.avoiding)
		}
	}
}
