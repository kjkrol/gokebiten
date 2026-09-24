package board

import (
	"fmt"
	"github.com/kjkrol/gram/plugin/host"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/render"
	"github.com/kjkrol/uid"
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

// Plugin wires a Board into a Game; it depends on world, and on collision only WithCollision.
type Plugin struct {
	Res Resources

	occupancy Occupancy
	renderer  *Renderer
	kinds     *cellKindDict
	seeded    *Layout

	worldPlugin *world.Plugin
	collision   *collision.Plugin
	module      *module
	standing    host.EachHost[Standing]
	body        plugin.Tag[Family]
}

var _ plugin.Plugin = (*Plugin)(nil)
var _ plugin.Populator = (*Plugin)(nil)

// NewPlugin builds a board over grid with the given occupancy cap, slowing worldPlugin's entities.
func NewPlugin(grid Grid, occupancy Occupancy, worldPlugin *world.Plugin) *Plugin {
	terrain := NewTerrainMap()
	p := &Plugin{
		occupancy:   occupancy,
		worldPlugin: worldPlugin,
		kinds:       newCellKindDict(),
	}
	p.Res.Logic.Board = NewBoard(grid, terrain)
	if ws, ok := p.Res.Logic.Board.Grid.(wrapSetter); ok {
		edges := worldPlugin.Res.Config.Space.Edges
		ws.SetWrap(edges.WrapsX(), edges.WrapsY())
	}
	if err := worldPlugin.RegisterBehavior(terrainSpeed(grid, terrain)); err != nil {
		panic(err)
	}
	return p
}

// =================================================================
// plugin.Plugin contract
// =================================================================

func (p *Plugin) Name() string { return "gram.board" }

// Install wires the standing report and, WithCollision, the terrain bodies.
func (p *Plugin) Install(ctx plugin.Installer) error {
	p.module = &module{
		cells:    newCellEntitySystem(p.Res.Logic.Board, p.worldPlugin, p.worldPlugin.Kinds().Reserve("board.cell")),
		standing: newStandingSystem(p.Res.Logic.Board, &p.standing),
	}
	if p.collision != nil {
		typeID := p.worldPlugin.Kinds().Reserve("board.terrain")
		p.body = p.worldPlugin.Kinds().DefineTag[Family]("board.body")
		p.module.bodies = newTerrainBodySystem(p.Res.Logic.Board, p.worldPlugin, typeID, p.body)
	}
	ctx.UseModule(p.module)
	return nil
}

// RunPlan rebuilds the terrain bodies after a terrain change and reports where everyone stands;
// call it after collision's RunPlan.
func (p *Plugin) RunPlan(ctx goke.RunCtx, d time.Duration) { p.module.RunPlan(ctx, d) }

// WithRenderer builds the board renderer, drawing each cell's CellKind.SpriteID from atlas.
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

// RegisterBehavior hosts an Each or Every of Standing, run every tick for every entity on the board;
// register before Use.
func (p *Plugin) RegisterBehavior(behaviors ...plugin.Behavior) error {
	for _, b := range behaviors {
		if err := p.standing.Add(b); err != nil {
			return fmt.Errorf("%w in %s — it takes Each for Standing", err, p.Name())
		}
	}
	return nil
}

// =================================================================
// board-specific
// =================================================================

// WithCollision makes terrain physical: every run of Solid cells becomes an immovable body in the
// world, pushed against by c and cutting sight, and every run of veiled cells a body carrying a
// vision.Transparency of 1 - Veil, dimming it. Call before Use.
func (p *Plugin) WithCollision(c *collision.Plugin) *Plugin {
	if c == nil {
		panic("board: WithCollision needs the collision plugin")
	}
	p.collision = c
	return p
}

// CellEntity is the entity standing for cell c — found, or spawned over the cell with a [Cell]
// and a [Ground] — so an effect cast on it is an effect on the cell's terrain.
func (p *Plugin) CellEntity(c CellID) uid.UID64 { return p.module.cells.entity(c) }

// DropCellEntity despawns a cell entity, and does nothing for any other entity; the terrain keeps
// what its Ground last said. The board does it itself once the entity's last effect ended.
func (p *Plugin) DropCellEntity(cb *goke.CmdBuf, id uid.UID64) { p.module.cells.drop(cb, id) }

// Body is the tag every terrain body carries, in board's tag Family; zero without WithCollision.
func (p *Plugin) Body() plugin.Tag[Family] { return p.body }

// Occupancy returns the occupancy tracker this plugin was built with.
func (p *Plugin) Occupancy() Occupancy { return p.occupancy }

// CellKindDict returns this Plugin's registered CellKinds.
func (p *Plugin) CellKindDict() CellKindDict { return p.kinds }

// Seed sets the terrain applied when this Stage starts fresh — see Populate.
func (p *Plugin) Seed(layout Layout) { p.seeded = &layout }

// Populate applies the seeded Layout, changing nothing and erroring on an unknown kind name.
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
