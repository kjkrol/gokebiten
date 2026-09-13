package board

import (
	"fmt"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
)

// Resources is board's single published Resources — Logic is what
// board-modifying code needs; Render is nil unless WithRenderer was called.
type Resources struct {
	Logic struct {
		Board *Board
	}
	Render *RenderState
}

// Persisted returns terrain for Persistence.Save/Load to include automatically.
func (r *Resources) Persisted() []any { return []any{r.Logic.Board.TerrainMap} }

var _ plugin.Serializable = (*Resources)(nil)

// Plugin wires a Board into a Game — depends only on world (for
// Position/Velocity/SpeedModifier). See plugins/navigation for entity
// movement/pathfinding built on top of this Board.
type Plugin struct {
	Res Resources

	terrainSpeed *TerrainSpeedModifier
	occupancy    Occupancy
	renderer     *Renderer
	kinds        *cellKindDict
	seeded       *Layout

	worldPlugin *world.Plugin
}

var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.Populator = (*Plugin)(nil)

// NewPlugin builds a board over grid, capping cell occupancy per occupancy.
// worldPlugin is where the board's TerrainSpeedModifier registers itself.
// Register terrain kinds afterward via CellKindDict().Create.
func NewPlugin(grid Grid, occupancy Occupancy, worldPlugin *world.Plugin) *Plugin {
	terrain := NewTerrainMap()
	p := &Plugin{
		terrainSpeed: NewTerrainSpeedModifier(grid, terrain),
		occupancy:    occupancy,
		worldPlugin:  worldPlugin,
		kinds:        newCellKindDict(),
	}
	p.Res.Logic.Board = NewBoard(grid, terrain)
	if ts, ok := p.Res.Logic.Board.Grid.(toroidalSetter); ok {
		ts.SetToroidal(worldPlugin.Res.Config.Space.Toroidal)
	}
	worldPlugin.RegisterSpeedModifier(p.terrainSpeed)
	return p
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gokebiten.board" }

// Install is a no-op — board has no ECS wiring of its own; see plugins/navigation.
func (p *Plugin) Install(ctx plugin.Installer) error { return nil }

// RunPlan is a no-op — board has no per-tick work of its own; see plugins/navigation.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) {}

// WithRenderer builds this plugin's own board renderer, drawing each cell's CellKind.SpriteID from atlas.
func (p *Plugin) WithRenderer(atlas render.AtlasSource) {
	p.Res.Render = &RenderState{ShowGridLines: true}
	p.renderer = newRenderer(p.worldPlugin.Camera(), p.Res.Logic.Board, atlas, p.Res.Render)
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

// Serializable returns board's persistable state (its terrain).
func (p *Plugin) Serializable() plugin.Serializable { return &p.Res }

// =================================================================
// board-specific
// =================================================================

// Occupancy returns the occupancy tracker this plugin was built with.
func (p *Plugin) Occupancy() Occupancy { return p.occupancy }

// CellKindDict returns this Plugin's registered set of CellKinds — call
// Create to register kinds, Get/All to read them back.
func (p *Plugin) CellKindDict() CellKindDict { return p.kinds }

// Seed sets the terrain applied when this Stage starts fresh — see Populate.
func (p *Plugin) Seed(layout Layout) { p.seeded = &layout }

// Populate applies the seeded Layout, erroring (and changing nothing) on a kind name CellKindDict doesn't know.
func (p *Plugin) Populate() error {
	if p.seeded == nil {
		return nil
	}
	resolve := func(name string) (CellKind, error) {
		kind, ok := p.kinds.Get(name)
		if !ok {
			return CellKind{}, fmt.Errorf("board: unknown CellKind %q", name)
		}
		return kind, nil
	}

	var def CellKind
	if p.seeded.Default != "" {
		kind, err := resolve(p.seeded.Default)
		if err != nil {
			return err
		}
		def = kind
	}
	cells := make([]CellKind, len(p.seeded.Cells))
	for i, e := range p.seeded.Cells {
		kind, err := resolve(e.Kind)
		if err != nil {
			return err
		}
		cells[i] = kind
	}

	brd := p.Res.Logic.Board
	if p.seeded.Default != "" {
		brd.SetAll(def)
	}
	for i, e := range p.seeded.Cells {
		brd.Set(e.Cell, cells[i])
	}
	p.seeded = nil
	return nil
}
