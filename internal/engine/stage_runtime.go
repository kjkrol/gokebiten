package engine

import (
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// stageRuntime is one active Stage: its own ecsHost, the mandatory
// world.Plugin auto-installed into it, and every Scene's Layers() already
// resolved into concrete render.Renderer values (built once, at entry —
// Draw just replays them, keyed by Scene.Name() to match
// Stage.Stack().Composition().Order()).
type stageRuntime struct {
	host        *ecsHost
	stage       game.Stage
	world       *world.Plugin
	sceneLayers map[string][]render.Renderer
}

// enterStage builds a Stage's world from scratch: a fresh *goke.ECS, the
// mandatory world.Plugin, Stage.Init/Restore/Spawn, every Scene's
// renderers, and the single ecs.Setup flush — mirroring what Engine.Init
// used to do once for the whole game, now repeatable per Stage.
func (e *Engine) enterStage(stage game.Stage) (*stageRuntime, error) {
	host := newECSHost()

	cfg := e.props.World
	if cfg.Camera.ViewportWidth == 0 && cfg.Camera.ViewportHeight == 0 {
		cfg.Camera.ViewportWidth = uint32(e.props.ScreenWidth)
		cfg.Camera.ViewportHeight = uint32(e.props.ScreenHeight)
	}
	worldPlugin := world.NewPlugin(cfg)

	ctx := &initializer{host: host, world: worldPlugin, tps: e.tps}
	if err := ctx.useBuiltin(worldPlugin); err != nil {
		return nil, err
	}
	if err := stage.Init(ctx); err != nil {
		return nil, err
	}

	restored, err := stage.Restore(&persistence{host: host})
	if err != nil {
		return nil, err
	}
	if !restored {
		if err := stage.Spawn(); err != nil {
			return nil, err
		}
		if err := host.runPopulate(); err != nil {
			return nil, err
		}
	}

	host.ecs.SetPlan(stage.Update)

	sceneLayers := make(map[string][]render.Renderer)
	for _, sc := range stage.Stack().All() {
		var layers []render.Renderer
		for _, factory := range sc.Layers() {
			layers = append(layers, host.registerRenderer(factory))
		}
		sceneLayers[sc.Name()] = layers
	}

	host.flushPendingSetup()

	return &stageRuntime{host: host, stage: stage, world: worldPlugin, sceneLayers: sceneLayers}, nil
}
