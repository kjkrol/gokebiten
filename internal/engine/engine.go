package engine

import (
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/render"
)

const (
	defaultTargetTPS = 60
)

// Engine drives a user-implemented game.Game — a collection of Stages —
// through the Ebitengine loop, one active Stage (and its own *goke.ECS) at
// a time. game.Game is the only container of Stages; Engine never caches
// its own copy — it calls game.Stages() whenever it needs to resolve a
// name (Init, and each SwitchStage).
type Engine struct {
	game    game.Game
	current *stageRuntime

	ticks       int
	step        time.Duration
	timeTracker *tracker
	props       game.Props
	inputs      *control.InputEvents
	tps         *game.TPS
	controller  *DefaultController

	// pendingSwitch names the Stage a SwitchStage call asked to enter,
	// picked up at the top of the next Update — see SwitchStage. Its
	// presence also tells Draw to show transitionOverlay for that one
	// frame instead of the (about to be replaced) current Stage.
	pendingSwitch     string
	transitionOverlay render.SolidBackground

	quit bool
}

var _ ebiten.Game = (*Engine)(nil)
var _ game.Runtime = (*Engine)(nil)

// NewEngine builds an Engine driving g — g.Props() supplies the window/
// tick-rate/world config.
func NewEngine(g game.Game) *Engine {
	props := g.Props()
	inputs := &control.InputEvents{}
	tps := &game.TPS{}

	targetTPS := defaultTargetTPS
	if props.TargetTPS != 0 {
		targetTPS = props.TargetTPS
	}
	controller := NewDefaultController(&DesktopAdapter{}, inputs)
	e := &Engine{
		game:              g,
		props:             props,
		inputs:            inputs,
		tps:               tps,
		step:              time.Second / time.Duration(targetTPS),
		timeTracker:       newTracker(),
		controller:        controller,
		transitionOverlay: render.SolidBackground{Color: color.RGBA{A: 255}},
	}
	controller.SetHandler(HandlerFn(e.dispatchEvents))
	return e
}

// TPS returns the engine's measured-ticks-per-second counter.
func (e *Engine) TPS() *game.TPS { return e.tps }

// Persistence returns the active Stage's Save/Load/List surface.
func (e *Engine) Persistence() game.Persistence { return &persistence{host: e.current.host} }

func (e *Engine) Paused() bool { return e.current.host.ecs.Paused() }

func (e *Engine) Pause() { e.current.host.ecs.Pause() }

func (e *Engine) Resume() { e.current.host.ecs.Resume() }

func (e *Engine) TogglePause() {
	if e.current.host.ecs.Paused() {
		e.current.host.ecs.Resume()
	} else {
		e.current.host.ecs.Pause()
	}
}

// Camera returns the active Stage's built-in world's shared Camera.
func (e *Engine) Camera() camera.Camera { return e.current.world.Camera() }

// Quit ends the Ebitengine loop after this tick.
func (e *Engine) Quit() { e.quit = true }

// SwitchStage requests a transition to the Stage named name — resolved
// fresh against game.Stages() (never a cached copy), performed
// synchronously at the start of the next Update, after this tick shows
// transitionOverlay for one frame.
func (e *Engine) SwitchStage(name string) error {
	stages, _ := e.game.Stages()
	if _, ok := stages[name]; !ok {
		return fmt.Errorf("gokebiten: unknown stage %q", name)
	}
	e.pendingSwitch = name
	return nil
}

// Init calls Game.Stages and enters the initial Stage — split out from Run
// so tests can exercise it without starting the (blocking) Ebitengine loop.
func (e *Engine) Init() error {
	stages, initial := e.game.Stages()
	stage, ok := stages[initial]
	if !ok {
		return fmt.Errorf("gokebiten: initial stage %q not found among registered Stages", initial)
	}

	current, err := e.enterStage(stage)
	if err != nil {
		return err
	}
	e.current = current
	return nil
}

// Run calls Init, then starts the Ebitengine loop.
func (e *Engine) Run() {
	if err := e.Init(); err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowSize(e.props.ScreenWidth, e.props.ScreenHeight)
	ebiten.SetWindowTitle(e.props.Title)
	if err := ebiten.RunGame(e); err != nil {
		log.Fatal(err)
	}
}

// =================================================================
// ebiten.Game contract implementation
// =================================================================

func (e *Engine) Update() error {
	if e.quit {
		return ebiten.Termination
	}

	if e.pendingSwitch != "" {
		name := e.pendingSwitch
		e.pendingSwitch = ""
		stages, _ := e.game.Stages()
		stage, ok := stages[name]
		if !ok {
			return fmt.Errorf("gokebiten: unknown stage %q", name)
		}
		current, err := e.enterStage(stage)
		if err != nil {
			return err
		}
		e.current = current
	}

	e.controller.Capture(e.inputs)
	e.controller.Update(nil, 0)
	e.inputs.ResetTransient()

	if e.current.host.ecs.Paused() {
		return nil
	}

	steps := e.timeTracker.calculateSteps(e.step, 5)
	for range steps {
		e.current.host.ecs.Tick(e.step)
		e.ticks++
	}

	if e.timeTracker.processStatsInterval() {
		e.tps.Ticks = e.ticks
		e.ticks = 0
	}

	return nil
}

func (e *Engine) Draw(screen *ebiten.Image) {
	if e.pendingSwitch != "" {
		e.transitionOverlay.Draw(screen)
		return
	}
	for _, name := range e.current.stage.Composition().Order() {
		for _, r := range e.current.sceneLayers[name] {
			r.Draw(screen)
		}
	}
}

func (e *Engine) Layout(outsideWidth, outsideHeight int) (int, int) {
	return e.props.ScreenWidth, e.props.ScreenHeight
}

// =================================================================

// dispatchEvents is the controller's single registered handler — it runs
// exclusively the active Scene's HandleEvents (Composition.Active()).
// Stage has no HandleEvents of its own: input is the Scene's sole
// responsibility.
func (e *Engine) dispatchEvents(events *control.InputEvents) {
	stage := e.current.stage
	comp := stage.Composition()
	active := comp.Active()
	if active == "" {
		return
	}
	if sc, ok := stage.Stack().Get(active); ok {
		sc.HandleEvents(events, e, comp)
	}
}
