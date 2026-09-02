package board

import (
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Plugin wires a Board into a Game — depends only on world (for
// Position/Velocity/SpeedModifier). See plugins/navigation for entity
// movement/pathfinding built on top of this Board.
type Plugin struct {
	board        *Board
	terrain      *TerrainMap
	terrainSpeed *TerrainSpeedModifier
	occupancy    Occupancy
	renderer     *Renderer
}

var _ gokebiten.Plugin = (*Plugin)(nil)
var _ gokebiten.Saveable = (*Plugin)(nil)

// NewPlugin builds a board over grid, capping cell occupancy per occupancy.
func NewPlugin(grid Grid, occupancy Occupancy) *Plugin {
	terrain := NewTerrainMap()
	return &Plugin{
		board:        NewBoard(grid, terrain),
		terrain:      terrain,
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
	ctx.Provide(p.board)
	ctx.Provide(p.terrain)
	ctx.Provide(p)
	worldPlugin.RegisterSpeedModifier(p.terrainSpeed)
	return nil
}

// Board returns the underlying Board (Grid + Terrain), built at construction.
func (p *Plugin) Board() *Board { return p.board }

// Occupancy returns the occupancy tracker this plugin was built with.
func (p *Plugin) Occupancy() Occupancy { return p.occupancy }

// Terrain returns the underlying TerrainMap, built at construction — mutate it directly (Set/SetMany/SetAll) to shape the map.
func (p *Plugin) Terrain() *TerrainMap { return p.terrain }

// SaveTargets returns terrain for Persistence.Save/Load to include automatically.
func (p *Plugin) SaveTargets() []any { return []any{p.terrain} }

// WithRenderer builds this plugin's own board renderer (cellSize world-pixels per cell, style picks the sprite to draw).
func (p *Plugin) WithRenderer(cellSize float32, atlas render.AtlasSource, style CellStyle) *Plugin {
	p.renderer = newRenderer(p.board, cellSize, atlas, style)
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

// RunPlan is a no-op — board has no per-tick work of its own; see plugins/navigation.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {}

// EventHandler always returns nil — board has no input handling of its own; see plugins/navigation.
func (p *Plugin) EventHandler() control.EventHandler { return nil }
