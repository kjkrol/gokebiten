package gokebiten

import (
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/internal/timing"
	"github.com/kjkrol/gokebiten/render"
)

const (
	defaultTargetTPS = 60
)

type GameProps struct {
	Title                     string
	TargetTPS                 int
	ScreenWidth, ScreenHeight int
}

// TPS is the built-in measured-ticks-per-second counter, inserted by NewGame.
type TPS struct{ Ticks int }

type Game struct {
	Persistence *Persistence

	ticks         int
	step          time.Duration
	timeTracker   *timing.Tracker
	resources     *Resources
	props         *GameProps
	inputs        *control.InputEvents
	tps           *TPS
	ecs           *goke.ECS
	renderSeq     []render.Renderer
	controller    *control.DefaultController
	pluginManager *pluginManager
	pendingSetup  []func() []goke.System
}

var _ ebiten.Game = (*Game)(nil)

// NewGame builds a Game, pre-populating its resource registry with *GameProps, *control.InputEvents, and *TPS.
func NewGame(props *GameProps) *Game {
	resources := NewResources()
	inputs := &control.InputEvents{}
	tps := &TPS{}
	resources.insertResource(props)
	resources.insertResource(inputs)
	resources.insertResource(tps)

	targetTPS := defaultTargetTPS
	if props != nil && props.TargetTPS != 0 {
		targetTPS = props.TargetTPS
	}
	controller := control.NewDefaultController(&control.DesktopAdapter{}, inputs)
	game := &Game{
		resources:   resources,
		props:       props,
		inputs:      inputs,
		tps:         tps,
		step:        time.Second / time.Duration(targetTPS),
		timeTracker: timing.New(),
		ecs:         goke.New(),
		controller:  controller,
	}
	game.Persistence = &Persistence{game: game}
	game.pluginManager = &pluginManager{game: game}
	return game
}

// Resources returns the game's shared resource registry.
func (g *Game) Resources() *Resources { return g.resources }

// EventHandler sets the handler Update calls once per tick with this tick's input events.
func (g *Game) EventHandler(handler control.EventHandler) {
	g.controller.SetHandler(handler)
}

// EventHandlerFn is EventHandler for a plain closure, letting call sites skip declaring a named type.
func (g *Game) EventHandlerFn(fn func(events *control.InputEvents)) {
	g.EventHandler(eventHandlerFunc(fn))
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
	g.controller.Capture(g.inputs)
	g.controller.Update(nil, 0)
	g.inputs.ResetTransient()

	if g.ecs.Paused() {
		return nil
	}

	steps := g.timeTracker.CalculateSteps(g.step, 5)
	for range steps {
		g.ecs.Tick(g.step)
		g.ticks++
	}

	if g.timeTracker.ProcessStatsInterval() {
		g.tps.Ticks = g.ticks
		g.ticks = 0
		g.resources.forEach(func(v any) {
			if r, ok := v.(Resettable); ok {
				r.Reset()
			}
		})
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, sys := range g.renderSeq {
		sys.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.props.ScreenWidth, g.props.ScreenHeight
}

func (g *Game) Run() {
	if err := g.pluginManager.finalizePending(); err != nil {
		log.Fatal(err)
	}
	g.flushPendingSetup()

	ebiten.SetWindowSize(g.props.ScreenWidth, g.props.ScreenHeight)
	ebiten.SetWindowTitle(g.props.Title)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// RegComp registers ECS component type C.
func RegComp[C any](ctx *GameCtx) goke.CompID {
	return ctx.game.ecs.RegComp[C]()
}

func (g *Game) regSys(factory func() goke.System) goke.Runnable {
	return g.ecs.RegSys(factory())
}

// useModule defers m.RegSystems to the one-time Setup call and tracks m for Persistence.Load.
func (g *Game) useModule(m goke.Module) {
	g.pluginManager.track(m)
	sys := goke.SystemFn{OnInit: func(si *goke.SysInit) { m.RegSystems(g.ecs) }}
	g.pendingSetup = append(g.pendingSetup, func() []goke.System { return []goke.System{sys} })
}

// setup tracks each provider for Persistence.Load and defers its SetupSystems to the one-time ecs.Setup call.
func (g *Game) setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		g.pluginManager.track(p)
		g.pendingSetup = append(g.pendingSetup, p.SetupSystems)
	}
}

func (g *Game) registerRenderer(factory func() render.Renderer) render.Renderer {
	renderer := factory()
	sys := goke.SystemFn{OnInit: func(si *goke.SysInit) { renderer.Init(si) }}
	g.pendingSetup = append(g.pendingSetup, func() []goke.System { return []goke.System{sys} })
	return renderer
}

// flushPendingSetup evaluates every deferred producer once and runs the result through a single ecs.Setup call.
func (g *Game) flushPendingSetup() {
	if len(g.pendingSetup) == 0 {
		return
	}
	var systems []goke.System
	for _, produce := range g.pendingSetup {
		systems = append(systems, produce()...)
	}
	g.ecs.Setup(systems...)
	g.pendingSetup = nil
}

// eventHandlerFunc adapts a plain closure to control.EventHandler — see Game.EventHandlerFn.
type eventHandlerFunc func(events *control.InputEvents)

func (f eventHandlerFunc) HandleEvents(events *control.InputEvents) { f(events) }
