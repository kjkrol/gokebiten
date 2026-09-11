package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin moves entities along a MoveOrder's path across a board, re-pathing automatically
// when terrain along the route changes. WithCommands adds right-click move orders for
// Selected entities; WithRenderer draws the remaining route.
type Plugin struct {
	speed int32

	boardPlugin *board.Plugin
	worldPlugin *world.Plugin

	board  *board.Board
	module *module

	res *Resources

	rendererEnabled bool
	pathAtlas       render.AtlasSource
	pathSprites     PathSprites
	pathRenderer    *PathRenderer

	camera camera.Camera
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds a navigation plugin over boardPlugin/worldPlugin, moving
// entities at speed world-units/sec before scaling, using cam for
// screen<->world conversion (right-click targeting and route rendering).
func NewPlugin(speed int32, boardPlugin *board.Plugin, worldPlugin *world.Plugin, cam camera.Camera) *Plugin {
	return &Plugin{speed: speed, boardPlugin: boardPlugin, worldPlugin: worldPlugin, camera: cam}
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.navigation" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	brd := p.boardPlugin.Res.Logic.Board
	p.board = brd

	occupancy := p.boardPlugin.Occupancy()
	finder := newPathFinder(brd, brd, occupancy)
	navSys := newNavigationSystem(finder, brd, brd, occupancy, p.speed)
	navSys.BindSpace(p.worldPlugin.Space())

	if p.rendererEnabled {
		p.pathRenderer = NewPathRenderer(p.camera, brd, p.pathAtlas, p.pathSprites)
		p.pathRenderer.BindSpace(p.worldPlugin.Space())
	}
	p.res = &Resources{}
	moveCommandSystem := newMoveCommandSystem(finder, p.res)

	p.module = &module{navigationSystem: navSys, moveCommandSystem: moveCommandSystem}
	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs the navigation system (and the command system, if enabled) for this tick — call before world's own RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.module.RunPlan(ctx, d)
}

// WithRenderer builds this plugin's own PathRenderer, drawing the remaining route for every selected, en-route entity — call SetPathSprites first.
func (p *Plugin) WithRenderer(cam camera.Camera, atlas render.AtlasSource) {
	p.rendererEnabled = true
	p.pathAtlas = atlas
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.pathRenderer == nil {
		return nil
	}
	return p.pathRenderer
}

// EventHandler returns the default right-click move-order control.EventHandler, or nil unless WithCommands was called.
func (p *Plugin) EventHandler() control.EventHandler {
	return NewDefaultCommandEventHandler(p.board, p.camera, p.res)
}

// Serializable is a no-op — navigation has nothing to persist.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// =================================================================
// navigation-specific
// =================================================================

// SetPathSprites sets the sprite set WithRenderer's PathRenderer draws — call before UsePlugin.
func (p *Plugin) SetPathSprites(sprites PathSprites) *Plugin {
	p.pathSprites = sprites
	return p
}
