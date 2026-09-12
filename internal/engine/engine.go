package engine

import (
	"log"
	"reflect"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

const (
	defaultTargetTPS = 60
)

// Props configures Engine's window, target tick rate, and the built-in world.
type Props struct {
	Title                     string
	TargetTPS                 int
	ScreenWidth, ScreenHeight int
	World                     world.Config
}

// Engine drives a user-implemented game.Game through the Ebitengine loop.
type Engine struct {
	game  game.Game
	world *world.Plugin

	ticks       int
	step        time.Duration
	timeTracker *tracker
	resources   *storage
	props       *Props
	inputs      *control.InputEvents
	tps         *game.TPS
	ecs         *goke.ECS
	layers      []render.Renderer
	controller  *DefaultController

	tracked      []any
	pendingSetup []func() []goke.System
	names        map[string]bool

	quit bool
}

var _ ebiten.Game = (*Engine)(nil)
var _ game.Runtime = (*Engine)(nil)

// NewEngine builds an Engine driving g.
func NewEngine(props *Props, g game.Game) *Engine {
	inputs := &control.InputEvents{}
	tps := &game.TPS{}

	targetTPS := defaultTargetTPS
	if props != nil && props.TargetTPS != 0 {
		targetTPS = props.TargetTPS
	}
	controller := NewDefaultController(&DesktopAdapter{}, inputs)
	return &Engine{
		game:        g,
		resources:   newStorage(),
		props:       props,
		inputs:      inputs,
		tps:         tps,
		step:        time.Second / time.Duration(targetTPS),
		timeTracker: newTracker(),
		ecs:         goke.New(),
		controller:  controller,
	}
}

// TPS returns the engine's measured-ticks-per-second counter.
func (e *Engine) TPS() *game.TPS { return e.tps }

// Persistence returns the engine's Save/Load/List surface.
func (e *Engine) Persistence() game.Persistence { return &persistence{engine: e} }

func (e *Engine) Paused() bool { return e.ecs.Paused() }

func (e *Engine) Pause() { e.ecs.Pause() }

func (e *Engine) Resume() { e.ecs.Resume() }

func (e *Engine) TogglePause() {
	if e.ecs.Paused() {
		e.ecs.Resume()
	} else {
		e.ecs.Pause()
	}
}

// Camera returns the built-in world's shared Camera.
func (e *Engine) Camera() camera.Camera { return e.world.Camera() }

// Quit ends the Ebitengine loop after this tick.
func (e *Engine) Quit() { e.quit = true }

// Init calls Game.Init and flushes queued ECS setup — split out from Run so
// tests can exercise it without starting the (blocking) Ebitengine loop.
func (e *Engine) Init() error {
	ctx := &initializer{engine: e}
	cfg := e.props.World
	if cfg.Camera.ViewportWidth == 0 && cfg.Camera.ViewportHeight == 0 {
		cfg.Camera.ViewportWidth = uint32(e.props.ScreenWidth)
		cfg.Camera.ViewportHeight = uint32(e.props.ScreenHeight)
	}
	e.world = world.NewPlugin(cfg)
	if err := ctx.useBuiltin(e.world); err != nil {
		return err
	}
	if err := e.game.Init(ctx); err != nil {
		return err
	}
	restored, err := e.game.Restore(e.Persistence())
	if err != nil {
		return err
	}
	if !restored {
		batches, err := e.game.Spawn()
		if err != nil {
			return err
		}
		for _, b := range batches {
			e.world.Populate(b.Count, b.Spawner)
		}
	}

	e.ecs.SetPlan(e.game.Update)
	for _, factory := range e.game.Draw(e) {
		e.layers = append(e.layers, e.registerRenderer(factory))
	}
	e.controller.SetHandler(HandlerFn(func(events *control.InputEvents) {
		e.game.HandleEvents(events, e)
	}))

	e.flushPendingSetup()
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

	e.controller.Capture(e.inputs)
	e.controller.Update(nil, 0)
	e.inputs.ResetTransient()

	if e.ecs.Paused() {
		return nil
	}

	steps := e.timeTracker.calculateSteps(e.step, 5)
	for range steps {
		e.ecs.Tick(e.step)
		e.ticks++
	}

	if e.timeTracker.processStatsInterval() {
		e.tps.Ticks = e.ticks
		e.ticks = 0
	}

	return nil
}

func (e *Engine) Draw(screen *ebiten.Image) {
	for _, l := range e.layers {
		l.Draw(screen)
	}
}

func (e *Engine) Layout(outsideWidth, outsideHeight int) (int, int) {
	return e.props.ScreenWidth, e.props.ScreenHeight
}

// =================================================================

func (e *Engine) registerRenderer(factory func() render.Renderer) render.Renderer {
	r := factory()

	sys := goke.SystemFn{OnInit: func(si *goke.SysInit) { r.Init(si) }}
	e.addPendingSetup(func() []goke.System { return []goke.System{sys} })

	return r
}

// track records v so providedComps/postLoadSystems/saveTargets can find it later.
func (e *Engine) track(v any) { e.tracked = append(e.tracked, v) }

// addPendingSetup queues producer to run once, during Run's flush.
func (e *Engine) addPendingSetup(producer func() []goke.System) {
	e.pendingSetup = append(e.pendingSetup, producer)
}

// providedComps collects LoadComps from every tracked value implementing goke.CompProvider.
func (e *Engine) providedComps() []goke.CompToken { return goke.ProvidedComps(e.tracked...) }

// postLoadSystems collects PostLoad from every tracked value implementing PostLoader.
func (e *Engine) postLoadSystems() []goke.System {
	var systems []goke.System
	for _, v := range e.tracked {
		if pl, ok := v.(plugin.PostLoader); ok {
			systems = append(systems, pl.PostLoad())
		}
	}
	return systems
}

// runRestore calls Restore on every tracked value implementing Restorer,
// synchronously, right after Persistence.Load decodes their Persisted() pointers.
func (e *Engine) runRestore() {
	for _, v := range e.tracked {
		if r, ok := v.(plugin.Restorer); ok {
			r.Restore()
		}
	}
}

// saveTargets collects Persisted from every tracked value implementing
// Serializable, keyed by its Go type name (tracked values have no Plugin.Name()).
func (e *Engine) saveTargets() map[string][]any {
	out := make(map[string][]any)
	for _, v := range e.tracked {
		if s, ok := v.(plugin.Serializable); ok {
			out[reflect.TypeOf(v).String()] = s.Persisted()
		}
	}
	return out
}

// persistGroups combines tracked Serializables, plugin-published
// Serializables, and extra into one name-keyed map for save/load.
func (e *Engine) persistGroups(extra ...any) map[string][]any {
	groups := e.saveTargets()
	for name, targets := range e.resources.persisted() {
		groups[name] = targets
	}
	for _, r := range extra {
		groups[reflect.TypeOf(r).String()] = []any{r}
	}
	return groups
}

// flushPendingSetup evaluates every deferred producer once and runs the result through a single ecs.Setup call.
func (e *Engine) flushPendingSetup() {
	if len(e.pendingSetup) == 0 {
		return
	}
	var systems []goke.System
	for _, produce := range e.pendingSetup {
		systems = append(systems, produce()...)
	}
	e.ecs.Setup(systems...)
	e.pendingSetup = nil
}
