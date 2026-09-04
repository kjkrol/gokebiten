package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin moves entities along a MoveOrder's path across a board, re-pathing automatically
// when terrain along the route changes. WithCommands adds right-click move orders for
// Selected entities; WithRenderer draws the remaining route.
type Plugin struct {
	speed int32

	board  *board.Board
	module *module

	commandState *CommandState

	rendererEnabled bool
	pathAtlas       render.AtlasSource
	pathSprites     PathSprites
	pathRenderer    *PathRenderer

	camera render.Camera
}

var _ plugins.Plugin = (*Plugin)(nil)

// NewPlugin builds a navigation plugin, moving entities at speed world-units/sec before scaling.
func NewPlugin(speed int32) *Plugin {
	return &Plugin{speed: speed}
}

// =================================================================
// plugins.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.navigation" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	boardPlugin, err := ctx.Require[*board.Plugin]()
	if err != nil {
		return err
	}
	brd, err := ctx.Require[*board.Board]()
	if err != nil {
		return err
	}
	worldPlugin, err := ctx.Require[*world.Plugin]()
	if err != nil {
		return err
	}
	p.board = brd
	occupancy := boardPlugin.Occupancy()
	finder := newPathFinder(brd, brd, occupancy)
	navSys := newNavigationSystem(finder, brd, brd, occupancy, p.speed)
	navSys.BindSpace(worldPlugin.Space())

	camera, err := ctx.Require[render.Camera]()
	if err != nil {
		return err
	}
	p.camera = camera

	if p.rendererEnabled {
		p.pathRenderer = NewPathRenderer(brd, p.pathAtlas, p.pathSprites)
		p.pathRenderer.BindSpace(worldPlugin.Space())
	}
	p.commandState = &CommandState{}
	moveCommandSystem := newMoveCommandSystem(finder, p.commandState)
	ctx.Provide(p.commandState)

	p.module = &module{navigationSystem: navSys, moveCommandSystem: moveCommandSystem}
	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs the navigation system (and the command system, if enabled) for this tick — call before world's own RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.module.RunPlan(ctx, d)
}

// WithRenderer builds this plugin's own PathRenderer, drawing the remaining route for every selected, en-route entity — call SetPathSprites first.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
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
	return NewDefaultCommandEventHandler(p.board, p.camera, p.commandState)
}

// =================================================================
// navigation-specific
// =================================================================

// SetPathSprites sets the sprite set WithRenderer's PathRenderer draws — call before UsePlugin.
func (p *Plugin) SetPathSprites(sprites PathSprites) *Plugin {
	p.pathSprites = sprites
	return p
}
