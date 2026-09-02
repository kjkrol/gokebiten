package navigation

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/gokg/spatial"
)

func testSpace(t *testing.T) *gokg.Space {
	t.Helper()
	space, err := gokg.NewSpace(gokg.Config{
		Width: 1000, Height: 1000,
		BucketSize: spatial.ResolutionFrom(64), BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("gokg.NewSpace: %v", err)
	}
	return space
}

// pushOnce jumps every entity's Position to a fixed cell the first time
// it's armed and run — simulating a collision shove that bypasses
// world.MoveSystem, so NavigationSystem must notice the mismatch on its own.
type pushOnce struct {
	grid  board.Grid
	to    board.CellID
	size  uint32
	armed bool

	pos   goke.Comp[world.Position]
	query *goke.Query
}

func (p *pushOnce) Init(si *goke.SysInit) { p.query = si.NewQueryBuilder(&p.pos).Build() }

func (p *pushOnce) Update(_ *goke.CmdBuf, _ time.Duration) {
	if !p.armed {
		return
	}
	p.armed = false
	p.query.All()
	for p.query.Next() {
		positions := p.pos.Slice(p.query.Cursor())
		for i := range positions {
			positions[i] = world.Position{AABB: board.CellAABB(p.grid, p.to, p.size)}
		}
	}
}

func TestNavigationSystem_Update_DeviationTriggersRepath(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steer := newNavigationSystem(newPathFinder(grid, terrain, occupancy), grid, terrain, occupancy, 20)
	pusher := &pushOnce{grid: grid, size: 8}

	start, _ := grid.CellIndex(0, 0)
	target, _ := grid.CellIndex(4, 0)
	pushed, _ := grid.CellIndex(3, 0)
	pusher.to = pushed

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var order goke.Comp[MoveOrder]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &order)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0] = world.Position{AABB: board.CellAABB(grid, start, 8)}
		order.Slice(&f.Cursor)[0] = MoveOrder{Target: target}
		occupancy.Enter(start, id)

		q = si.NewQueryBuilder(&cell, &order).Build()
	}})

	pusherHandle := ecs.RegSys(pusher)
	steerHandle := ecs.RegSys(steer)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(pusherHandle, d)
		ctx.Run(steerHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second) // computes the initial start->target path

	c, mt := readCellAndMoveOrder(t, q, &cell, &order)
	if mt.Path.Length == 0 {
		t.Fatal("expected a path to have been computed on the first tick")
	}
	if c.ID != start {
		t.Fatalf("Cell.ID = %v, want %v (no movement yet, no deviation)", c.ID, start)
	}
	firstStep := mt.Path.Steps[0]
	if firstStep == pushed || firstStep == target {
		t.Fatalf("first path step %v should be an intermediate cell — grid too small for this test", firstStep)
	}

	pusher.armed = true
	ecs.Tick(time.Second) // pusher jumps the entity ahead; navigation must notice and re-path

	c, mt = readCellAndMoveOrder(t, q, &cell, &order)
	if c.ID != pushed {
		t.Errorf("Cell.ID = %v, want %v (deviation should resync bookkeeping to the actual cell)", c.ID, pushed)
	}
	if mt.Path.Length == 0 {
		t.Fatal("expected the path to have been recomputed after the deviation")
	}
	if mt.Path.Steps[mt.Path.Index] == firstStep {
		t.Error("recomputed path should route onward from the pushed-to cell, not resume the stale pre-deviation path")
	}
}

// TestNavigationSystem_Update_TransientFlankerCellDoesNotInvalidatePath guards
// against a real bug: sampling a diagonal move at discrete ticks routinely
// lands the entity's continuous position in one of the two cells flanking
// the corner for a tick, before it reaches the actually-planned cell — that
// is not a deviation and must not trigger a full re-path.
func TestNavigationSystem_Update_TransientFlankerCellDoesNotInvalidatePath(t *testing.T) {
	grid := board.NewSquareGrid(5, 5, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steer := newNavigationSystem(newPathFinder(grid, terrain, occupancy), grid, terrain, occupancy, 20)

	previous, _ := grid.CellIndex(0, 1)
	expected, _ := grid.CellIndex(1, 0) // diagonal neighbor of previous
	flanker, _ := grid.CellIndex(1, 1)  // flanks the previous->expected corner

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var order goke.Comp[MoveOrder]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &order)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: previous}
		// Position already sits inside the flanker cell — a normal artifact
		// of sampling a diagonal move at discrete ticks, not a deviation.
		pos.Slice(&f.Cursor)[0] = world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](11, 11), 8, 8)}
		var mt MoveOrder
		mt.Target = expected
		mt.Path.Steps[0] = expected
		mt.Path.Length = 1
		order.Slice(&f.Cursor)[0] = mt
		occupancy.Enter(previous, id)

		q = si.NewQueryBuilder(&cell, &order).Build()
	}})

	steerHandle := ecs.RegSys(steer)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(steerHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second)

	c, mt := readCellAndMoveOrder(t, q, &cell, &order)
	if c.ID != flanker {
		t.Fatalf("Cell.ID = %v, want %v (bookkeeping should still track the actual cell)", c.ID, flanker)
	}
	if mt.Path.Length != 1 || mt.Path.Steps[0] != expected {
		t.Errorf("Path = %+v, want unchanged (Length=1, Steps[0]=%v) — a transient flanker cell shouldn't invalidate the path", mt.Path, expected)
	}
}

// TestNavigationSystem_Update_ArrivalStopsEntity guards against a real bug: on
// arrival the "arrived" branch removed MoveOrder but never zeroed Velocity —
// since the entity never matches this system's query again once MoveOrder is
// gone, nothing ever stopped it, and it drifted off the board.
func TestNavigationSystem_Update_ArrivalStopsEntity(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steer := newNavigationSystem(newPathFinder(grid, terrain, occupancy), grid, terrain, occupancy, 20)

	start, _ := grid.CellIndex(2, 0)
	target := start // already at the target — arrives on the very first tick

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var order goke.Comp[MoveOrder]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &order)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0] = world.Position{AABB: board.CellAABB(grid, start, 8)}
		vel.Slice(&f.Cursor)[0] = world.Velocity{Dir: geom.NewVec[float64](1, 0), Value: 50} // was already moving in
		order.Slice(&f.Cursor)[0] = MoveOrder{Target: target}
		occupancy.Enter(start, id)

		q = si.NewQueryBuilder(&cell, &vel).Build()
	}})

	steerHandle := ecs.RegSys(steer)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(steerHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second)

	q.All()
	found := false
	for q.Next() {
		cur := q.Cursor()
		velocities := vel.Slice(cur)
		for i := range cur.IDs {
			found = true
			if velocities[i].Value != 0 {
				t.Errorf("Velocity.Value = %v, want 0 after arriving at target", velocities[i].Value)
			}
		}
	}
	if !found {
		t.Fatal("expected to find the seeded entity")
	}
}

// TestNavigationSystem_Update_ArrivalSnapsToCellCenter guards that an entity
// arriving off-center (a natural consequence of continuous movement crossing
// a cell boundary at an arbitrary point) ends up exactly centered — needed
// for a board game, where units are expected to sit precisely on a cell.
func TestNavigationSystem_Update_ArrivalSnapsToCellCenter(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steer := newNavigationSystem(newPathFinder(grid, terrain, occupancy), grid, terrain, occupancy, 20)
	space := testSpace(t)
	steer.BindSpace(space)

	target, _ := grid.CellIndex(2, 0)
	// Within arrivalEpsilon of the target cell's true center (25,5) for a 10px cell at column 2.
	offCenter := world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](20, 1), 8, 8)}

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var order goke.Comp[MoveOrder]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &order)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: target}
		pos.Slice(&f.Cursor)[0] = offCenter
		order.Slice(&f.Cursor)[0] = MoveOrder{Target: target}
		occupancy.Enter(target, id)
		space.Insert(id, offCenter.AABB)
		space.Flush(nil)

		q = si.NewQueryBuilder(&pos).Build()
	}})

	steerHandle := ecs.RegSys(steer)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(steerHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second)

	want := grid.CellCenter(target)
	q.All()
	found := false
	for q.Next() {
		cur := q.Cursor()
		positions := pos.Slice(cur)
		for i := range cur.IDs {
			found = true
			p := positions[i]
			gotX := float64(p.TopLeft.X) + float64(p.Size.X)/2
			gotY := float64(p.TopLeft.Y) + float64(p.Size.Y)/2
			if gotX != want.X || gotY != want.Y {
				t.Errorf("center = (%v, %v), want (%v, %v) — should snap exactly to the cell center on arrival", gotX, gotY, want.X, want.Y)
			}
		}
	}
	if !found {
		t.Fatal("expected to find the seeded entity")
	}
}

// TestNavigationSystem_Update_ArrivalGlidesSmoothlyToCellCenter guards against
// a real bug: switching waypoints on cell-boundary crossing (rather than
// proximity to the cell's true center) made a large final correction happen
// in a single tick, visibly popping the entity into place. Arrival must
// glide in bounded steps and still land exactly on center.
func TestNavigationSystem_Update_ArrivalGlidesSmoothlyToCellCenter(t *testing.T) {
	grid := board.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steer := newNavigationSystem(newPathFinder(grid, terrain, occupancy), grid, terrain, occupancy, 20)
	space := testSpace(t)
	steer.BindSpace(space)

	target, _ := grid.CellIndex(2, 0)
	// Off the true center (25,5), within the target cell, well beyond arrivalEpsilon.
	offCenter := world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](17, 1), 8, 8)}

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var order goke.Comp[MoveOrder]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &order)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: target}
		pos.Slice(&f.Cursor)[0] = offCenter
		order.Slice(&f.Cursor)[0] = MoveOrder{Target: target}
		occupancy.Enter(target, id)
		space.Insert(id, offCenter.AABB)
		space.Flush(nil)

		q = si.NewQueryBuilder(&pos).Build()
	}})

	steerHandle := ecs.RegSys(steer)
	moveHandle := ecs.RegSys(world.NewMoveSystem(space))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(steerHandle, d)
		ctx.Run(moveHandle, d)
		ctx.Sync()
	})

	readCenter := func() (float64, float64) {
		t.Helper()
		q.All()
		for q.Next() {
			cur := q.Cursor()
			positions := pos.Slice(cur)
			for i := range cur.IDs {
				p := positions[i]
				return float64(p.TopLeft.X) + float64(p.Size.X)/2, float64(p.TopLeft.Y) + float64(p.Size.Y)/2
			}
		}
		t.Fatal("expected to find the seeded entity")
		return 0, 0
	}

	prevX, prevY := readCenter()
	maxStep := 0.0
	for range 200 {
		ecs.Tick(20 * time.Millisecond)
		x, y := readCenter()
		if step := math.Hypot(x-prevX, y-prevY); step > maxStep {
			maxStep = step
		}
		prevX, prevY = x, y
	}

	if maxStep > 3 {
		t.Errorf("largest single-tick step = %.2f, want <= 3 (arrival must glide smoothly, not jump)", maxStep)
	}

	want := grid.CellCenter(target)
	if prevX != want.X || prevY != want.Y {
		t.Errorf("final center = (%v, %v), want (%v, %v)", prevX, prevY, want.X, want.Y)
	}
}

// TestNavigationSystem_Update_ReproducesBoardDemoWallScenario reproduces the
// board-demo wall scenario (24x16 grid, a full-height wall at column 12,
// a unit routing from the left side to the right side) tick by tick over
// simulated real time, to observe directly whether the entity's logical
// Cell ever lands inside the wall and whether its Path changes shape
// without any external deviation.
func TestNavigationSystem_Update_ReproducesBoardDemoWallScenario(t *testing.T) {
	const (
		gridWidth, gridHeight, cellSize = uint32(24), uint32(16), uint32(32)
		wallCol                         = uint32(12)
		entitySize                      = uint32(22)
		speed                           = int32(cellSize * 2)
	)
	grid := board.NewSquareGrid(gridWidth, gridHeight, cellSize)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	var wallCells []board.CellID
	for y := uint32(2); y < gridHeight; y++ {
		c, _ := grid.CellIndex(wallCol, y)
		wallCells = append(wallCells, c)
	}
	terrain.SetMany(wallCells, board.CellKind{Cost: 1, Passable: false})
	occupancy := &board.SingleOccupancy{}

	start, _ := grid.CellIndex(2, 4)
	target, _ := grid.CellIndex(gridWidth-3, 4)

	pathFinder := newPathFinder(grid, terrain, occupancy)
	steer := newNavigationSystem(pathFinder, grid, terrain, occupancy, speed)
	space := testSpace(t)
	steer.BindSpace(space)

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var order goke.Comp[MoveOrder]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &order)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		startPos := world.Position{AABB: board.CellAABB(grid, start, entitySize)}
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0] = startPos
		order.Slice(&f.Cursor)[0] = MoveOrder{Target: target}
		occupancy.Enter(start, id)
		space.Insert(id, startPos.AABB)
		space.Flush(nil)

		q = si.NewQueryBuilder(&cell, &pos, &order).Build()
	}})

	steerHandle := ecs.RegSys(steer)
	moveHandle := ecs.RegSys(world.NewMoveSystem(space))
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(steerHandle, d)
		ctx.Run(moveHandle, d)
		ctx.Sync()
	})

	dt := time.Second / 60
	var lastSteps []board.CellID
	var prevCellID board.CellID
	var prevExpected board.CellID
	var havePrev bool
	replans := 0
	for tick := 0; tick < 60*15; tick++ {
		ecs.Tick(dt)

		q.All()
		for q.Next() {
			cur := q.Cursor()
			cells := cell.Slice(cur)
			positions := pos.Slice(cur)
			orders := order.Slice(cur)
			for i := range cur.IDs {
				if !terrain.Kind(cells[i].ID).Passable {
					t.Fatalf("tick %d: entity's logical Cell is inside impassable terrain: %v", tick, cells[i].ID)
				}
				path := orders[i].Path
				steps := append([]board.CellID(nil), path.Steps[:path.Length]...)
				if !equalSteps(steps, lastSteps) {
					replans++
					if havePrev {
						t.Logf("tick %d: INVALIDATED — actual(new cell)=%v, prevCell=%v, prevExpected=%v, pos=%v",
							tick, cells[i].ID, prevCellID, prevExpected, positions[i].TopLeft)
					}
					t.Logf("tick %d: path changed (len %d -> %d) steps=%v", tick, len(lastSteps), len(steps), steps)
					lastSteps = steps
				}
				prevCellID = cells[i].ID
				if path.Length > 0 && path.Index < path.Length {
					prevExpected = path.Steps[path.Index]
				}
				havePrev = true
			}
		}
	}
	t.Logf("total path changes observed: %d", replans)
	if replans != 1 {
		t.Errorf("total path changes = %d, want 1 (only the initial FindPath — no spurious mid-route invalidation)", replans)
	}
}

func equalSteps(a, b []board.CellID) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func readCellAndMoveOrder(t *testing.T, q *goke.Query, cell *goke.Comp[board.Cell], order *goke.Comp[MoveOrder]) (board.Cell, MoveOrder) {
	t.Helper()
	q.All()
	for q.Next() {
		cur := q.Cursor()
		cells := cell.Slice(cur)
		orders := order.Slice(cur)
		if len(cells) > 0 {
			return cells[0], orders[0]
		}
	}
	t.Fatal("expected to find the seeded entity")
	return board.Cell{}, MoveOrder{}
}
