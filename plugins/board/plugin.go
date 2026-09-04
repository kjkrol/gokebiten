package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires a Board into a Game — depends only on world (for
// Position/Velocity/SpeedModifier). See plugins/navigation for entity
// movement/pathfinding built on top of this Board.
type Plugin struct {
	board        *Board
	kinds        CellKindDict
	terrainSpeed *TerrainSpeedModifier
	occupancy    Occupancy
	renderer     *Renderer
	renderState  *RenderState

	worldPlugin *world.Plugin
}

var _ plugins.Plugin = (*Plugin)(nil)
var _ gokebiten.Saveable = (*Plugin)(nil)

// NewPlugin builds a board over grid, capping cell occupancy per occupancy
// and publishing kinds to Resources for board-modifying code to pick from.
// worldPlugin is where the board's TerrainSpeedModifier registers itself.
func NewPlugin(grid Grid, occupancy Occupancy, kinds CellKindDict, worldPlugin *world.Plugin) *Plugin {
	terrain := NewTerrainMap()
	return &Plugin{
		board:        NewBoard(grid, terrain),
		kinds:        kinds,
		terrainSpeed: NewTerrainSpeedModifier(grid, terrain),
		occupancy:    occupancy,
		worldPlugin:  worldPlugin,
	}
}

// =================================================================
// plugins.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.board" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	if err := ctx.RequirePlugin(p.worldPlugin); err != nil {
		return err
	}
	worldConfig, err := ctx.Require[world.Config]()
	if err != nil {
		return err
	}
	if ts, ok := p.board.Grid.(toroidalSetter); ok {
		ts.SetToroidal(worldConfig.Space.Toroidal)
	}
	ctx.Provide(p.board)
	ctx.Provide(p.kinds)
	if p.renderState != nil {
		ctx.Provide(p.renderState)
	}
	p.worldPlugin.RegisterSpeedModifier(p.terrainSpeed)
	return nil
}

// RunPlan is a no-op — board has no per-tick work of its own; see plugins/navigation.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {}

// WithRenderer builds this plugin's own board renderer, drawing each cell's CellKind.SpriteID from atlas.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.renderState = &RenderState{ShowGridLines: true}
	p.renderer = newRenderer(p.board, atlas, p.renderState)
}

// Renderer returns this plugin's own render.Renderer, or nil unless WithRenderer was called.
func (p *Plugin) Renderer() render.Renderer {
	if p.renderer == nil {
		return nil
	}
	return p.renderer
}

// EventHandler always returns nil — board has no input handling of its own; see plugins/navigation.
func (p *Plugin) EventHandler() control.EventHandler { return nil }

// =================================================================
// board-specific
// =================================================================

// Occupancy returns the occupancy tracker this plugin was built with.
func (p *Plugin) Occupancy() Occupancy { return p.occupancy }

// SaveTargets returns terrain for Persistence.Save/Load to include automatically.
func (p *Plugin) SaveTargets() []any { return []any{p.board.TerrainMap} }
