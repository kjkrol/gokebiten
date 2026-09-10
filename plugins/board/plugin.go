package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokebiten/resources"
)

// Resources is board's single published Resources — Logic is what
// board-modifying code needs; Render is nil unless WithRenderer was called.
type Resources struct {
	Logic struct {
		Board *Board
		Kinds CellKindDict
	}
	Render *RenderState
}

func (*Resources) Resources() {}

// Persisted returns terrain for Persistence.Save/Load to include automatically.
func (r *Resources) Persisted() []any { return []any{r.Logic.Board.TerrainMap} }

var _ resources.Resources = (*Resources)(nil)
var _ resources.Serializable = (*Resources)(nil)

// Plugin wires a Board into a Game — depends only on world (for
// Position/Velocity/SpeedModifier). See plugins/navigation for entity
// movement/pathfinding built on top of this Board.
type Plugin struct {
	res Resources

	terrainSpeed *TerrainSpeedModifier
	occupancy    Occupancy
	renderer     *Renderer

	worldPlugin *world.Plugin
}

var _ plugins.Plugin = (*Plugin)(nil)

// NewPlugin builds a board over grid, capping cell occupancy per occupancy
// and publishing kinds via Resources for board-modifying code to pick from.
// worldPlugin is where the board's TerrainSpeedModifier registers itself.
func NewPlugin(grid Grid, occupancy Occupancy, kinds CellKindDict, worldPlugin *world.Plugin) *Plugin {
	terrain := NewTerrainMap()
	p := &Plugin{
		terrainSpeed: NewTerrainSpeedModifier(grid, terrain),
		occupancy:    occupancy,
		worldPlugin:  worldPlugin,
	}
	p.res.Logic.Board = NewBoard(grid, terrain)
	p.res.Logic.Kinds = kinds
	return p
}

// =================================================================
// plugins.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.board" }

func (p *Plugin) Install(ctx *plugins.GameCtx) error {
	if err := ctx.RequirePlugin(p.worldPlugin); err != nil {
		return err
	}
	worldRes, err := ctx.Require[*world.Resources]()
	if err != nil {
		return err
	}
	if ts, ok := p.res.Logic.Board.Grid.(toroidalSetter); ok {
		ts.SetToroidal(worldRes.Config.Space.Toroidal)
	}
	p.worldPlugin.RegisterSpeedModifier(p.terrainSpeed)
	return nil
}

// RunPlan is a no-op — board has no per-tick work of its own; see plugins/navigation.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {}

// WithRenderer builds this plugin's own board renderer, drawing each cell's CellKind.SpriteID from atlas.
func (p *Plugin) WithRenderer(cam camera.Camera, atlas render.AtlasSource) {
	p.res.Render = &RenderState{ShowGridLines: true}
	p.renderer = newRenderer(cam, p.res.Logic.Board, atlas, p.res.Render)
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

// Resources returns board's single published Resources.
func (p *Plugin) Resources() resources.Resources { return &p.res }

// =================================================================
// board-specific
// =================================================================

// Occupancy returns the occupancy tracker this plugin was built with.
func (p *Plugin) Occupancy() Occupancy { return p.occupancy }
