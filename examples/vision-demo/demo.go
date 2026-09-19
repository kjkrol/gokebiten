// Command vision-demo shows entities steering around each other by sight
// rather than bouncing off each other by contact.
//
// Every entity carries a Sight cone, drawn on screen, and the flee behavior
// turns it away from whatever the cone picks up. Collisions are installed
// alongside and counted, so the telemetry line puts a number on how much the
// avoidance is buying: press A to switch it off and watch the counter climb.
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
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/collisions/strategies/elastic"
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
	hunterSpeed  = roamSpeed * 0.9
	preyKind     = "prey"
	hunterKind   = "hunter"
	backdropGrey = 40
)

// =========================== Game ===========================

// Demo is the vision demo — exactly one Stage.
type Demo struct{ stage *mainStage }

var _ game.Game = (*Demo)(nil)

func NewDemo() *Demo { return &Demo{stage: &mainStage{avoiding: true}} }

func (d *Demo) Props() game.Props {
	return game.Props{
		Title:       "gokebiten — sight and avoidance",
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
				k.Const(elastic.Bouncy{})),
		}
	})
	kinds.Define(hunterKind, func(k world.Kind[body]) world.EntKind {
		return world.EntKind{
			Position: k.Load(func(b body) world.Position { return b.pos }),
			Velocity: k.Load(func(b body) world.Velocity { return b.vel }),
			Components: append(s.sees(k),
				k.Const(world.Steering{Reflex: 1, TurnRate: 0.30}),
				k.Const(hunt.Predator{}), k.Const(collisions.Sensor{})),
		}
	})

	s.avoidance = flee.New()
	s.world.RegisterBehavior(s.avoidance)
	s.world.RegisterBehavior(hunt.New())
	s.world.RegisterBehavior(&faceTravel{})
	s.world.RegisterBehavior(elastic.New())
	s.world.RegisterBehavior(stats.New(&s.hits))
	s.world.RegisterBehavior(&eat{world: s.world})

	s.vision = vision.NewPlugin(s.world)
	s.collisions = collisions.NewPlugin(s.world)

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
		k.Const(vision.Sighted{}),
		k.Const(vision.SightOutline{}),
		collisions.Collidable(s.world.Space()),
		k.Const(collisions.Contacts{}),
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

// faceTravel points each entity's Sight where it is actually going, so the cone
// follows the swerve instead of staring at where the entity set off.
type faceTravel struct {
	query *goke.Query
	sight goke.Comp[vision.Sight]
	vel   goke.Comp[world.Velocity]
}

var _ world.Behavior = (*faceTravel)(nil)

func (f *faceTravel) Init(si *goke.SysInit) {
	f.query = si.NewQueryBuilder(&f.sight, &f.vel).Build()
}

func (f *faceTravel) Update(*goke.CmdBuf, time.Duration) {
	f.query.All()
	for f.query.Next() {
		cursor := f.query.Cursor()
		sights := f.sight.Slice(cursor)
		vels := f.vel.Slice(cursor)
		for i := range cursor.IDs {
			if d := vels[i].Dir; d.X != 0 || d.Y != 0 {
				sights[i].Facing = d
			}
		}
	}
}

// eat is what a catch means in this demo: the hunter touches a prey, and the
// prey is gone. This is the collision reaction the engine deliberately leaves
// to the game — Contacts names who was struck, world.Despawn takes it out.
type eat struct {
	world    *world.Plugin
	query    *goke.Query
	contacts goke.Comp[collisions.Contacts]
}

var _ world.Behavior = (*eat)(nil)

func (e *eat) Init(si *goke.SysInit) {
	e.query = si.NewQueryBuilder(&e.contacts).Include(goke.Include[hunt.Predator]()).Build()
}

func (e *eat) Update(cb *goke.CmdBuf, _ time.Duration) {
	e.query.All()
	for e.query.Next() {
		cursor := e.query.Cursor()
		contacts := e.contacts.Slice(cursor)
		for i := range cursor.IDs {
			for _, caught := range contacts[i].All() {
				e.world.Despawn(cb, caught.Other)
			}
		}
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
