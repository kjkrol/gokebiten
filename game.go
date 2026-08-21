package gokebiten

import (
	"io"
	"log"
	"os"
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
	SaveState(w io.Writer) error
	LoadState(r io.Reader) error
}

type Game struct {
	ticks       int
	step        time.Duration
	timeTracker *TimeTracker
	resources   resources
	ecs         *goke.ECS
	renderSeq   []render.Renderer
	controller  *control.DefaultController
	// pendingSetup collects everything that needs SysInit-gated construction
	// (renderer Init, module RegSystems, one-time world seeding) across
	// RenderSequence/UseModule/Setup/Load calls — ecs.Setup is callable only
	// once, so it all runs together, lazily, right before the game loop
	// starts (see Run).
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

func (g *Game) Step() time.Duration { return g.step }

// UseModule defers m.RegSystems to the same Setup call RenderSequence/Setup
// use, in call order — call it after any spatial.WorldModule that seeds
// entities m needs to already exist (e.g. a physics module scanning for
// Sensor tags at registration).
func (g *Game) UseModule(m goke.Module) {
	g.pendingSetup = append(g.pendingSetup, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		m.RegSystems(g.ecs)
	}})
}

// Setup defers each provider's SetupSystems to the same one-time ecs.Setup
// call RenderSequence/UseModule feed — in call order, mirroring ecs.Setup's
// own name and one-time-seeding spirit at the Game level.
func (g *Game) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		g.pendingSetup = append(g.pendingSetup, p.SetupSystems()...)
	}
}

// Save pauses the ECS, writes its snapshot to basePath+".ecs", and — if
// State (S in Resources[S, T]) implements encoding.BinaryMarshaler — writes
// it to basePath+".state". Resumes before returning either way.
func (g *Game) Save(basePath string) error {
	g.ecs.Pause()
	defer g.ecs.Resume()

	if err := g.ecs.Save(basePath + ".ecs"); err != nil {
		return err
	}
	f, err := os.Create(basePath + ".state")
	if err != nil {
		return err
	}
	defer f.Close()
	return g.resources.SaveState(f)
}

// Load restores a snapshot written by Save into this (must be freshly
// constructed) Game: the ECS snapshot, State if present, and — since space
// (the gokg spatial index) isn't part of the ECS snapshot — a rebuild of
// space from every loaded entity's kinematics.Position, deferred to the
// same Setup phase RenderSequence/UseModule/Setup use. onLoaded, if not
// nil, is called once that rebuild completes, with the loaded entity count
// (e.g. to set a telemetry field) — nil if you don't need it. providers
// supplies component tokens the same way ProvidedComps does (e.g. your
// physics module); render.Appearance is always included.
func (g *Game) Load(basePath string, space *gokg.Space, onLoaded func(count int), providers ...any) error {
	comps := append(goke.ProvidedComps(providers...), goke.LoadComp[render.Appearance]())
	if err := g.ecs.Load(basePath+".ecs", comps...); err != nil {
		return err
	}
	if f, err := os.Open(basePath + ".state"); err == nil {
		defer f.Close()
		if err := g.resources.LoadState(f); err != nil {
			return err
		}
	}

	g.pendingSetup = append(g.pendingSetup, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var pos goke.Comp[kinematics.Position]
		query := si.NewQueryBuilder(&pos).Build()
		query.All()
		count := 0
		for query.Next() {
			cursor := query.Cursor()
			positions := pos.Slice(cursor)
			for i, id := range cursor.IDs {
				space.Insert(id, positions[i].AABB)
			}
			count += len(cursor.IDs)
		}
		space.Flush(nil)
		if onLoaded != nil {
			onLoaded(count)
		}
	}})
	return nil
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
