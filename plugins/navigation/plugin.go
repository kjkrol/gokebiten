package navigation

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires a navigationSystem (and, with WithCommands, a commandSystem)
// into a Game — depends on plugins/board (for Grid/Terrain) and plugins/world
// (for Position/Velocity/Space). WithCommands additionally depends on
// plugins/selection (Selected).
type Plugin struct {
	speed int32

	board      *board.Board
	pathFinder *pathFinder
	navigation *navigationSystem

	commandsEnabled bool
	commandState    *CommandState
	commands        *commandSystem

	rendererEnabled bool
	pathRenderer    *PathRenderer

	camera render.Camera
}

var _ gokebiten.Plugin = (*Plugin)(nil)

// NewPlugin builds a navigation plugin, moving entities at speed world-units/sec before scaling.
func NewPlugin(speed int32) *Plugin {
	return &Plugin{speed: speed}
}

func (p *Plugin) Name() string { return "gokebiten.navigation" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
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
	p.pathFinder = newPathFinder(brd, brd, occupancy)
	p.navigation = newNavigationSystem(p.pathFinder, brd, brd, occupancy, p.speed)
	p.navigation.BindSpace(worldPlugin.World().Space())

	if p.commandsEnabled || p.rendererEnabled {
		camera, err := ctx.Require[render.Camera]()
		if err != nil {
			return err
		}
		p.camera = camera
	}
	if p.rendererEnabled {
		p.pathRenderer = NewPathRenderer(brd)
	}
	if p.commandsEnabled {
		p.commandState = &CommandState{}
		p.commands = newCommandSystem(p.pathFinder, p.commandState)
		ctx.Provide(p.commandState)
		ctx.UseModule(p.commands)
	}

	ctx.UseModule(p.navigation)
	return nil
}

// WithCommands enables right-click move orders for Selected, en-route entities.
func (p *Plugin) WithCommands() *Plugin {
	p.commandsEnabled = true
	return p
}

// EventHandler returns the default right-click move-order control.EventHandler, or nil unless WithCommands was called.
func (p *Plugin) EventHandler() control.EventHandler {
	if p.commands == nil {
		return nil
	}
	return NewDefaultCommandEventHandler(p.board, p.camera, p.commandState)
}

// WithRenderer builds this plugin's own PathRenderer, drawing the remaining route for every selected, en-route entity.
func (p *Plugin) WithRenderer() *Plugin {
	p.rendererEnabled = true
	return p
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.pathRenderer == nil {
		return nil
	}
	return p.pathRenderer
}

// RunPlan runs the navigation system (and the command system, if enabled) for this tick — call before world's own RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.navigation.RunPlan(ctx, d)
	if p.commands != nil {
		p.commands.RunPlan(ctx, d)
	}
}
