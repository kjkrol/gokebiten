package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires a Board and its SteeringSystem into a Game — depends only on
// world (for Position/Velocity/SpeedModifier), never on collisions.
type Plugin struct {
	board        *Board
	steering     *SteeringSystem
	terrainSpeed *TerrainSpeedModifier
	renderer     *Renderer
	commands     *CommandSystem
	commandState *CommandState
	occupancy    Occupancy
	camera       render.Camera
}

var _ gokebiten.Plugin = (*Plugin)(nil)

// NewPlugin builds a board over grid/terrain, steering occupancy-tracked entities at speed world-units/sec before scaling.
func NewPlugin(grid Grid, terrain Terrain, occupancy Occupancy, speed int32) *Plugin {
	return &Plugin{
		board:        NewBoard(grid, terrain),
		steering:     NewSteeringSystem(grid, terrain, occupancy, speed),
		terrainSpeed: NewTerrainSpeedModifier(grid, terrain),
		occupancy:    occupancy,
	}
}

func (p *Plugin) Name() string { return "gokebiten.board" }

func (p *Plugin) Install(ctx *gokebiten.GameCtx) error {
	worldPlugin, err := ctx.Require[*world.Plugin]()
	if err != nil {
		return err
	}
	p.steering.BindSpace(worldPlugin.World().Space())
	if p.renderer != nil || p.commands != nil {
		camera, err := ctx.Require[render.Camera]()
		if err != nil {
			return err
		}
		p.camera = camera
	}
	ctx.Provide(p.board)
	ctx.UseModule(p.steering)
	if p.commands != nil {
		ctx.Provide(p.commandState)
		ctx.UseModule(p.commands)
	}
	worldPlugin.RegisterSpeedModifier(p.terrainSpeed)
	return nil
}

// Board returns the underlying Board (Grid + Terrain), built at construction.
func (p *Plugin) Board() *Board { return p.board }

// WithRenderer builds this plugin's own board renderer (cellSize world-pixels per cell, style picks fill color).
func (p *Plugin) WithRenderer(cellSize float32, style CellStyle) *Plugin {
	p.renderer = newRenderer(p.board, cellSize, style)
	return p
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// CellRenderer returns the concrete board renderer for further interaction (SetShowGridLines), or nil.
func (p *Plugin) CellRenderer() *Renderer { return p.renderer }

// WithCommands enables right-click move orders for Selected, en-route entities.
func (p *Plugin) WithCommands() *Plugin {
	p.commandState = &CommandState{}
	p.commands = NewCommandSystem(p.board, p.board, p.occupancy, p.commandState)
	return p
}

// Commands returns the command system, or nil unless WithCommands was called.
func (p *Plugin) Commands() *CommandSystem { return p.commands }

// EventHandler returns the default right-click move-order control.EventHandler, or nil unless WithCommands was called.
func (p *Plugin) EventHandler() control.EventHandler {
	if p.commands == nil {
		return nil
	}
	return NewDefaultCommandEventHandler(p.board, p.camera, p.commandState)
}

// RunPlan runs the steering system (and the command system, if enabled) for this tick — call before world's own RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.steering.RunPlan(ctx, d)
	if p.commands != nil {
		p.commands.RunPlan(ctx, d)
	}
}
