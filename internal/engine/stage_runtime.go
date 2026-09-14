package engine

import (
	"github.com/kjkrol/gokebiten/game"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// stageRuntime is one active Stage: its own ecsHost, the world.Plugin the
// Stage installed via UseWorld (nil if it didn't), and every Scene's Layers() already
// resolved into concrete render.Renderer values (built once, at entry —
// Draw just replays them, keyed by Scene.Name() to match
// Stage.Stack().Composition().Order()).
type stageRuntime struct {
	host        *ecsHost
	stage       game.Stage
	world       *world.Plugin
	sceneLayers map[string][]render.Renderer
}

// enterStage builds a Stage's world from scratch: a fresh *goke.ECS,
// Stage.Init (which may install a world.Plugin via UseWorld), Restore/Spawn, every Scene's
// renderers, and the single ecs.Setup flush — mirroring what Engine.Init
// used to do once for the whole game, now repeatable per Stage.
func (e *Engine) enterStage(stage game.Stage) (*stageRuntime, error) {
	host := newECSHost()

	ctx := &initializer{host: host, tps: e.tps, screenWidth: e.props.ScreenWidth, screenHeight: e.props.ScreenHeight}
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

	return &stageRuntime{host: host, stage: stage, world: ctx.world, sceneLayers: sceneLayers}, nil
}
