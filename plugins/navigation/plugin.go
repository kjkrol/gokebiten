package navigation

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/selection"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/render"
)

// Plugin moves entities along a MoveOrder's path across a board, re-pathing when terrain changes.
// WithCommands adds right-click move orders; WithRenderer draws the remaining route.
type Plugin struct {
	boardPlugin *board.Plugin
	worldPlugin *world.Plugin
	selected    plugin.Tag[selection.Family]

	board  *board.Board
	module *module

	res    *Resources
	finder *pathFinder

	pathSprites  PathSprites
	pathRenderer *PathRenderer

	camera camera.Camera
}

var _ plugin.Plugin = (*Plugin)(nil)

// NewPlugin builds a navigation plugin over a board; entities move as their Steering profile says.
func NewPlugin(boardPlugin *board.Plugin, worldPlugin *world.Plugin, selectionPlugin *selection.Plugin) *Plugin {
	return &Plugin{boardPlugin: boardPlugin, worldPlugin: worldPlugin, camera: worldPlugin.Camera(), selected: selectionPlugin.Tags().Selected}
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gram.navigation" }

func (p *Plugin) Install(ctx plugin.Installer) error {
	brd := p.boardPlugin.Res.Logic.Board
	p.board = brd

	occupancy := p.boardPlugin.Occupancy()
	finder := newPathFinder(brd, brd, occupancy)
	p.finder = finder
	if p.pathRenderer != nil {
		p.pathRenderer.finder = finder
	}
	navSys := newNavigationSystem(finder, brd, brd, occupancy)
	navSys.BindSpace(p.worldPlugin.Space())

	p.res = &Resources{}
	moveCommandSystem := newMoveCommandSystem(finder, p.res, p.selected)

	p.module = &module{navigationSystem: navSys, moveCommandSystem: moveCommandSystem}
	ctx.UseModule(p.module)
	return nil
}

// RunPlan runs navigation, and commands if enabled, for this tick; call before world's RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {
	p.module.RunPlan(ctx, d)
}

// WithRenderer draws the remaining route of every selected entity; call SetPathSprites first.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.pathRenderer = NewPathRenderer(p.camera, p.board, atlas, p.pathSprites, p.selected)
	p.pathRenderer.BindSpace(p.worldPlugin.Space())
	p.pathRenderer.finder = p.finder
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.pathRenderer == nil {
		return nil
	}
	return p.pathRenderer
}

// EventHandler returns the right-click move-order handler, or nil without WithCommands.
func (p *Plugin) EventHandler() control.EventHandler {
	return NewDefaultCommandEventHandler(p.board, p.camera, p.res)
}

// Serializable is a no-op — navigation has nothing to persist.
func (p *Plugin) Serializable() plugin.Serializable { return nil }

// RegisterBehavior reports ErrUnhostedBehavior — navigation hosts no behaviors.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	for _, b := range behaviors {
		return fmt.Errorf("%w: %T in %s", plugin.ErrUnhostedBehavior, b, p.Name())
	}
	return nil
}

// =================================================================
// navigation-specific
// =================================================================

// SetPathSprites sets the sprite set WithRenderer's PathRenderer draws — call before UsePlugin.
func (p *Plugin) SetPathSprites(sprites PathSprites) *Plugin {
	p.pathSprites = sprites
	return p
}
