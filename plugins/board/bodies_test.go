package board_test

import (
	"maps"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/collision"
	"github.com/kjkrol/gram/plugins/vision"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/uid"
)

// installCtx is the plugin.Installer a Stage would hand over, minus the engine.
type installCtx struct {
	ecs     *goke.ECS
	pending []func() []goke.System
}

func (c *installCtx) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(*goke.SysInit) { m.RegSystems(c.ecs) }}
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}
func (c *installCtx) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.pending = append(c.pending, p.SetupSystems)
	}
}
func (c *installCtx) RegSys(factory func() goke.System) goke.Runnable { return c.ecs.RegSys(factory()) }
func (c *installCtx) ECS() *goke.ECS                                  { return c.ecs }

// mover is a unit's row: where it starts, where it keeps driving when heading is set, what it
// sees, and how it moves (zero: Land).
type mover struct {
	cell    board.CellID
	heading geom.Vec
	sight   *vision.Sight
	domain  board.Domain
	offset  float64 // shifts the box right, to straddle two cells
}

const unitSize = 22

// bodiesWorld is world + collision + board WithCollision (+ vision) ticked as the demo ticks them.
type bodiesWorld struct {
	t     *testing.T
	w     *world.Plugin
	brd   *board.Plugin
	ecs   *goke.ECS
	base  goke.Comp[world.Base]
	body  goke.OptComp[plugin.Tags[board.Family]]
	sight goke.OptComp[vision.Sight]
	tau   goke.OptComp[vision.Transparency]
	q     *goke.Query
}

func newBodiesWorld(t *testing.T, grid board.Grid, width, height uint32, terrain func(*board.Board), units []mover, behaviors ...plugin.Behavior) *bodiesWorld {
	t.Helper()
	bw := &bodiesWorld{t: t}
	bw.w = world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: width, Height: height},
		Entities: world.EntitiesCfg{MaxCount: 16, MinSize: unitSize, MaxSize: unitSize},
	})
	c := collision.NewPlugin(bw.w)
	bw.brd = board.NewPlugin(grid, &board.MultipleOccupancy{}, bw.w).WithCollision(c)
	terrain(bw.brd.Res.Logic.Board)
	if err := bw.brd.RegisterBehavior(behaviors...); err != nil {
		t.Fatal(err)
	}
	v := vision.NewPlugin(bw.w)

	ctx := &installCtx{ecs: goke.New()}
	if err := bw.w.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := bw.brd.Install(ctx); err != nil {
		t.Fatal(err)
	}
	if err := v.Install(ctx); err != nil {
		t.Fatal(err)
	}

	for i, u := range units {
		spec := kind.Spec{
			kind.Load(func(m mover) world.Position {
				box := board.CellAABB(grid, m.cell, unitSize)
				box.TopLeft.X += m.offset
				box.BottomRight.X += m.offset
				return world.Position{AABB: box}
			}),
			kind.Const(world.Velocity{}),
			kind.Const(collision.Collider{}),
			kind.Const(collision.Physics{}),
			kind.Load(func(m mover) board.Cell { return board.Cell{ID: m.cell} }),
			kind.Load(func(m mover) board.Mover {
				if m.domain == 0 {
					return board.Mover{Domain: board.Land}
				}
				return board.Mover{Domain: m.domain}
			}),
		}
		if u.heading != (geom.Vec{}) {
			spec = append(spec, kind.Load(func(m mover) world.Steering {
				return world.Steering{Want: m.heading, WantSpeed: 64, MaxSpeed: 64}
			}))
		}
		if u.sight != nil {
			spec = append(spec, kind.Const(*u.sight))
		}
		name := string(rune('a' + i))
		bw.w.Seed(kind.Define[mover](bw.w.Kinds(), name, spec).Entry(u))
	}
	if err := bw.w.Populate(); err != nil {
		t.Fatal(err)
	}

	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		bw.q = si.NewQueryBuilder(&bw.base).Optional(&bw.body).Optional(&bw.sight).Optional(&bw.tau).Build()
	}})
	ctx.ecs.Setup(systems...)
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		bw.w.RunPlan(rc, d)
		c.RunPlan(rc, d)
		bw.brd.RunPlan(rc, d)
		v.RunPlan(rc, d)
		rc.Sync()
	})
	bw.ecs = ctx.ecs
	return bw
}

func (bw *bodiesWorld) tick() { bw.ecs.Tick(time.Second / 60) }

// snapshot lists the bodies' boxes and the units' boxes, units by TypeID order.
func (bw *bodiesWorld) snapshot() (bodies, units []geom.AABB) {
	units = make([]geom.AABB, 0)
	byType := map[kind.ID]geom.AABB{}
	for bw.q.All(); bw.q.Next(); {
		cur := bw.q.Cursor()
		bases := bw.base.Slice(cur)
		bodyMarks := bw.body.Slice(cur)
		for i := range cur.IDs {
			if bodyMarks != nil && bodyMarks[i].Has(bw.brd.Body()) {
				bodies = append(bodies, bases[i].Pos.AABB.AABB)
			} else {
				byType[bases[i].TypeID] = bases[i].Pos.AABB.AABB
			}
		}
	}
	types := slices.Sorted(maps.Keys(byType))
	for _, id := range types {
		units = append(units, byType[id])
	}
	return bodies, units
}

// isBody reports whether id is one of the terrain bodies.
func (bw *bodiesWorld) isBody(id uid.UID64) bool {
	for bw.q.All(); bw.q.Next(); {
		cur := bw.q.Cursor()
		if m := bw.body.Slice(cur); m != nil && slices.Contains(cur.IDs, id) {
			return m[slices.Index(cur.IDs, id)].Has(bw.brd.Body())
		}
	}
	return false
}

// seen returns what the one observer saw.
func (bw *bodiesWorld) seen() (vision.Sighted, bool) {
	for bw.q.All(); bw.q.Next(); {
		cur := bw.q.Cursor()
		if !bw.sight.Present(cur) {
			continue
		}
		return bw.sight.Slice(cur)[0].Seen, true
	}
	return vision.Sighted{}, false
}

// overlaps reports whether a and b share interior, not just an edge.
func overlaps(a, b geom.AABB) bool {
	const eps = 1e-6
	return min(a.BottomRight.X, b.BottomRight.X)-max(a.TopLeft.X, b.TopLeft.X) > eps &&
		min(a.BottomRight.Y, b.BottomRight.Y)-max(a.TopLeft.Y, b.TopLeft.Y) > eps
}

func (bw *bodiesWorld) assertClear(tick int, units, bodies []geom.AABB) {
	bw.t.Helper()
	for i, u := range units {
		for _, b := range bodies {
			if overlaps(u, b) {
				bw.t.Fatalf("tick %d: unit %d at %v overlaps body %v", tick, i, u, b)
			}
		}
	}
}

const cellSize = 32

func squareWorld(t *testing.T, units ...mover) (*bodiesWorld, board.CellID) {
	t.Helper()
	return squareWorldWith(t, nil, units...)
}

// squareWorldWith is squareWorld with a behavior registered on the board.
func squareWorldWith(t *testing.T, behavior plugin.Behavior, units ...mover) (*bodiesWorld, board.CellID) {
	t.Helper()
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	for i := range units {
		if units[i].cell == 0 {
			units[i].cell = cell(1, 7)
		}
	}
	var behaviors []plugin.Behavior
	if behavior != nil {
		behaviors = append(behaviors, behavior)
	}
	bw := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, func(brd *board.Board) {
		brd.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
		for y := uint32(1); y <= 14; y++ {
			brd.Set(cell(3, y), board.CellKind{Name: "wall", Cost: 1, Solid: true})
		}
	}, units, behaviors...)
	return bw, cell(3, 7)
}

var east = geom.NewVec(1, 0)

func TestBodies_AWallColumnIsOneBodyAUnitCannotEnter(t *testing.T) {
	bw, _ := squareWorld(t, mover{heading: east})
	bodies, units := bw.snapshot()
	if len(bodies) != 1 {
		t.Fatalf("%d bodies, want 1", len(bodies))
	}
	wall := bodies[0]
	for tick := range 60 {
		bw.tick()
		_, units = bw.snapshot()
		bw.assertClear(tick, units, bodies)
	}
	if units[0].BottomRight.X < wall.TopLeft.X-1 {
		t.Errorf("unit ends at %v, never reached the wall at %v", units[0], wall.TopLeft.X)
	}
}

func TestBodies_AUnitPushedByAnotherStaysOutOfTheWall(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	bw, _ := squareWorld(t, mover{cell: cell(2, 7)}, mover{cell: cell(1, 7), heading: east})
	bodies, _ := bw.snapshot()
	for tick := range 120 {
		bw.tick()
		_, units := bw.snapshot()
		bw.assertClear(tick, units, bodies)
	}
	_, units := bw.snapshot()
	if start := float64(cellSize) + (cellSize-unitSize)/2; units[1].TopLeft.X < start+5 {
		t.Errorf("the pusher never moved: %v behind %v", units[1], units[0])
	}
}

func TestBodies_AHexIsCoveredAndKeepsAUnitOut(t *testing.T) {
	grid := board.DefaultGrids{}.Hex(4, 4, cellSize)
	hex, _ := grid.CellIndex(1, 1)
	start, _ := grid.CellAt(geom.NewVec(20, grid.CellCenter(hex).Y))
	bw := newBodiesWorld(t, grid, 320, 256, func(brd *board.Board) {
		brd.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
		brd.Set(hex, board.CellKind{Name: "rock", Solid: true})
	}, []mover{{cell: start, heading: east}})
	bodies, _ := bw.snapshot()
	if len(bodies) != 2*board.HexCapStrips+1 {
		t.Fatalf("%d bodies for one hex, want %d", len(bodies), 2*board.HexCapStrips+1)
	}
	for tick := range 90 {
		bw.tick()
		_, units := bw.snapshot()
		bw.assertClear(tick, units, bodies)
	}
	_, units := bw.snapshot()
	center := grid.CellCenter(hex)
	if units[0].BottomRight.X < center.X-math.Sqrt(3)/2*cellSize-1 {
		t.Errorf("unit ends at %v, never reached the hex round %v", units[0], center)
	}
}

func TestBodies_FollowTheTerrainAsItChanges(t *testing.T) {
	bw, gap := squareWorld(t, mover{})
	bw.tick()
	bw.brd.Res.Logic.Board.Set(gap, board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
	bw.tick()
	bodies, units := bw.snapshot()
	if len(bodies) != 2 {
		t.Fatalf("%d bodies after opening a gap, want 2", len(bodies))
	}
	if got := bw.w.Res.Telemetry.Count; got != len(bodies)+len(units) {
		t.Errorf("telemetry counts %d entities, want %d", got, len(bodies)+len(units))
	}
}

func TestBodies_OccludeSight(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	observer := mover{cell: cell(1, 7), sight: &vision.Sight{Facing: east, HalfAngle: math.Pi / 8, Radius: 300}}
	target := mover{cell: cell(5, 7)}

	bw, _ := squareWorld(t, observer, target)
	bw.tick()
	if seen, ok := bw.seen(); !ok || seen.Count != 1 || !bw.isBody(seen.IDs[0]) {
		t.Errorf("saw %v through the wall, want the wall alone", seen.IDs[:seen.Count])
	}

	open := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, func(brd *board.Board) {
		brd.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
	}, []mover{observer, target})
	open.tick()
	if seen, ok := open.seen(); !ok || seen.Count != 1 {
		t.Errorf("saw %d across open ground, want 1", seen.Count)
	}
}

// forestColumn is grass with a forest down column 3, veiling sight by veil.
func forestColumn(grid board.Grid, veil float64) func(*board.Board) {
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	return func(brd *board.Board) {
		brd.SetAll(board.CellKind{Name: "grass", Cost: 1, Allows: board.Land})
		for y := uint32(1); y <= 14; y++ {
			brd.Set(cell(3, y), board.CellKind{Name: "forest", Cost: 1, Allows: board.Land, Veil: veil})
		}
	}
}

func TestBodies_AFullyVeiledCellOnlyBlocksSight(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	forest := forestColumn(grid, 1)
	observer := mover{cell: cell(1, 7), sight: &vision.Sight{Facing: east, HalfAngle: math.Pi / 8, Radius: 300}}
	target := mover{cell: cell(5, 7)}
	bw := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, forest, []mover{observer, target})
	bw.tick()
	if seen, ok := bw.seen(); !ok || seen.Count != 1 || !bw.isBody(seen.IDs[0]) {
		t.Errorf("saw %v through the forest, want the forest alone", seen.IDs[:seen.Count])
	}
	for bw.q.All(); bw.q.Next(); {
		cur := bw.q.Cursor()
		if m := bw.body.Slice(cur); m != nil {
			for i, b := range bw.base.Slice(cur) {
				if m[i].Has(bw.brd.Body()) && b.Caps&aabbworld.CanCollide != 0 {
					t.Errorf("the forest body carries caps %v, want none that collide", b.Caps)
				}
			}
		}
	}

	walker := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, forest, []mover{{cell: cell(1, 7), heading: east}})
	for range 90 {
		walker.tick()
	}
	if _, units := walker.snapshot(); units[0].TopLeft.X < float64(4*cellSize) {
		t.Errorf("unit ends at %v, want it past the forest column", units[0])
	}
}

// A forest column one cell (32) thick at Veil 0.6 costs 80 of reach: the target two cells past it
// is 165 away in budget terms and 117 as the crow flies.
func TestBodies_AVeilDimsSightByItsDepth(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	forest := forestColumn(grid, 0.6)
	target := mover{cell: cell(5, 7)}
	look := func(radius float64, clear bool) vision.Sighted {
		observer := mover{cell: cell(1, 7), sight: &vision.Sight{Facing: east, HalfAngle: math.Pi / 8, Radius: radius, Clear: clear}}
		bw := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, forest, []mover{observer, target})
		bw.tick()
		seen, ok := bw.seen()
		if !ok {
			t.Fatal("no observer")
		}
		for _, id := range seen.IDs[:seen.Count] {
			if bw.isBody(id) {
				t.Errorf("radius %v: the forest %d is listed as seen", radius, id)
			}
		}
		return seen
	}

	if seen := look(160, false); seen.Count != 0 {
		t.Errorf("at 160 through the forest saw %v, want nothing", seen.IDs[:seen.Count])
	}
	if seen := look(170, false); seen.Count != 1 {
		t.Errorf("at 170 through the forest saw %d, want the target", seen.Count)
	}
	if seen := look(160, true); seen.Count != 1 {
		t.Errorf("at 160 looking over the forest saw %d, want the target", seen.Count)
	}
}

func TestBodies_AWallCutsSightWhateverTheVeils(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	for _, clear := range []bool{false, true} {
		observer := mover{cell: cell(1, 7), sight: &vision.Sight{Facing: east, HalfAngle: math.Pi / 8, Radius: 300, Clear: clear}}
		bw, _ := squareWorld(t, observer, mover{cell: cell(5, 7)})
		bw.tick()
		if seen, ok := bw.seen(); !ok || seen.Count != 1 || !bw.isBody(seen.IDs[0]) {
			t.Errorf("clear %v: saw %v through the wall, want the wall alone", clear, seen.IDs[:seen.Count])
		}
	}
}

func TestBodies_VeiledBodiesCarryTheirTransparency(t *testing.T) {
	grid := board.DefaultGrids{}.Square(6, 16, cellSize)
	cell := func(x, y uint32) board.CellID { c, _ := grid.CellIndex(x, y); return c }
	bw := newBodiesWorld(t, grid, 6*cellSize, 16*cellSize, func(brd *board.Board) {
		forestColumn(grid, 0.6)(brd)
		brd.Set(cell(3, 0), board.CellKind{Name: "wall", Cost: 1, Solid: true})
	}, []mover{{cell: cell(1, 7)}})
	bw.tick()

	walls, forests, units := 0, 0, 0
	for bw.q.All(); bw.q.Next(); {
		cur := bw.q.Cursor()
		bases, marks := bw.base.Slice(cur), bw.body.Slice(cur)
		for i, id := range cur.IDs {
			var tau *vision.Transparency
			if bw.tau.Present(cur) {
				tau = &bw.tau.Slice(cur)[i]
			}
			switch {
			case marks == nil || !marks[i].Has(bw.brd.Body()):
				units++
				if tau != nil {
					t.Errorf("unit %d carries a Transparency", id)
				}
			case bases[i].Caps&aabbworld.CanCollide != 0:
				walls++
				if tau != nil {
					t.Errorf("wall %d carries a Transparency %v, want none: it cuts", id, tau.Value)
				}
			default:
				forests++
				if tau == nil || math.Abs(tau.Value-0.4) > 1e-9 {
					t.Errorf("forest %d carries %v, want a Transparency of 0.4", id, tau)
				}
			}
		}
	}
	if walls != 1 || forests < 1 || units != 1 {
		t.Errorf("found %d walls, %d forests, %d units; want 1, some, 1", walls, forests, units)
	}
}
