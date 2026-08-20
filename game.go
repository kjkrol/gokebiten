package gokebiten

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/physics/kinematics"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/spatial"
	"github.com/kjkrol/gokg"
)

const (
	defaultTargetTPS = 60
)

type GameProps struct {
	Title                     string
	TargetTPS                 int
	ScreenWidth, ScreenHeight int
}

// resources is the narrow contract Game itself needs — satisfied by
// *Resources[S, T] for any S, T. Unexported: Resources is the only, and only
// intended, implementation.
type resources interface {
	GetGameProps() *GameProps
	GetInputEvents() *control.InputEvents
	GetSpaceConfig() spatial.Config
	TPS() *int
	Reset()
}

type Game struct {
	ticks       int
	step        time.Duration
	timeTracker *TimeTracker
	resources   resources
	ecs         *goke.ECS
	renderSeq   []render.Renderer
	controller  *control.DefaultController
	spaceConfig spatial.Config
	world       *spatial.World
	// pendingSetup collects everything that needs SysInit-gated construction
	// (renderer Init, world population) across RenderSequence/Populate/
	// PopulateStatic calls — ecs.Setup is callable only once, so it all runs
	// together, lazily, right before the game loop starts (see Run).
	pendingSetup []goke.System
}

var _ ebiten.Game = (*Game)(nil)

func NewGame(res resources) *Game {
	targetTPS := defaultTargetTPS
	if res.GetGameProps() != nil && res.GetGameProps().TargetTPS != 0 {
		targetTPS = res.GetGameProps().TargetTPS
	}
	controller := control.NewDefaultController(&control.DesktopAdapter{}, res.GetInputEvents())
	game := &Game{
		resources:   res,
		step:        time.Second / time.Duration(targetTPS),
		timeTracker: NewTimeTracker(),
		ecs:         goke.New(),
		controller:  controller,
		spaceConfig: res.GetSpaceConfig(),
	}
	return game
}

func (g *Game) SetEventHandler(handler control.EventHandler) {
	g.controller.SetHandler(handler)
}

func (g *Game) Paused() bool { return g.ecs.Paused() }

func (g *Game) Pause() { g.ecs.Pause() }

func (g *Game) Resume() { g.ecs.Resume() }

func (g *Game) TogglePause() {
	if g.ecs.Paused() {
		g.ecs.Resume()
	} else {
		g.ecs.Pause()
	}
}

func RegComp[C any](game *Game) goke.CompID {
	return game.ecs.RegComp[C]()
}

func (g *Game) RegSys(factory func() goke.System) goke.Runnable {
	return g.ecs.RegSys(factory())
}

func (g *Game) ECS() *goke.ECS { return g.ecs }

func (g *Game) Space() *gokg.Space {
	if g.world == nil {
		panic("gokebiten: UseWorld must be called before Space")
	}
	return g.world.Space()
}

func (g *Game) Step() time.Duration { return g.step }

// UseModule defers m.RegSystems to the same Setup call Populate/RenderSequence
// use, in call order — call it after Populate if m needs entities to exist
// first.
func (g *Game) UseModule(m goke.Module) {
	g.pendingSetup = append(g.pendingSetup, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		m.RegSystems(si.ECS())
	}})
}

func (g *Game) registerRenderer(factory func() render.Renderer) render.Renderer {
	renderer := factory()
	g.pendingSetup = append(g.pendingSetup, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		renderer.Init(si)
	}})
	return renderer
}

func (g *Game) Loop(plan func(ctx goke.RunCtx, d time.Duration)) {
	g.ecs.SetPlan(plan)
}

func (g *Game) RenderSequence(rendererFactories ...func() render.Renderer) {
	for _, factory := range rendererFactories {
		renderer := g.registerRenderer(factory)
		g.renderSeq = append(g.renderSeq, renderer)
	}
}

// UseWorld builds the spatial index from this Game's SpaceConfig,
// provisioned for pop. Must be called before Populate, PopulateStatic or
// Space.
func (g *Game) UseWorld(pop spatial.Population) {
	g.world = spatial.NewWorld(g.spaceConfig, pop)
}

// Populate spawns count moving entities — see spatial.World.Populate.
// Requires UseWorld.
func (g *Game) Populate(count int, telemetry *kinematics.Telemetry, populators ...spatial.EntityExtras) {
	if g.world == nil {
		panic("gokebiten: UseWorld must be called before Populate")
	}
	g.pendingSetup = append(g.pendingSetup, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		g.world.Populate(si, count, telemetry, populators...)
	}})
}

// PopulateStatic spawns count static entities — see spatial.World.PopulateStatic.
// Requires UseWorld.
func (g *Game) PopulateStatic(count int, telemetry *kinematics.Telemetry, populators ...spatial.EntityExtras) {
	if g.world == nil {
		panic("gokebiten: UseWorld must be called before PopulateStatic")
	}
	g.pendingSetup = append(g.pendingSetup, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		g.world.PopulateStatic(si, count, telemetry, populators...)
	}})
}

func (g *Game) Update() error {
	g.controller.Capture(g.resources.GetInputEvents())
	g.controller.Update(nil, 0)
	g.resources.GetInputEvents().ResetTransient()

	if g.ecs.Paused() {
		return nil
	}

	steps := g.timeTracker.CalculateSteps(g.step, 5)
	for range steps {
		g.ecs.Tick(g.step)
		g.ticks++
	}

	if g.timeTracker.ProcessStatsInterval() {
		*g.resources.TPS() = g.ticks
		g.ticks = 0
		g.resources.Reset()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, sys := range g.renderSeq {
		sys.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	ScreenWidth := g.resources.GetGameProps().ScreenWidth
	ScreenHeight := g.resources.GetGameProps().ScreenHeight
	return ScreenWidth, ScreenHeight
}

func (g *Game) Run() {
	if len(g.pendingSetup) > 0 {
		g.ecs.Setup(g.pendingSetup...)
		g.pendingSetup = nil
	}

	ScreenWidth := g.resources.GetGameProps().ScreenWidth
	ScreenHeight := g.resources.GetGameProps().ScreenHeight
	Title := g.resources.GetGameProps().Title
	ebiten.SetWindowSize(ScreenWidth, ScreenHeight)
	ebiten.SetWindowTitle(Title)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
