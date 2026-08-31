package board_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/board/grids"
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
// world.MoveSystem, so SteeringSystem must notice the mismatch on its own.
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

func TestSteeringSystem_Update_DeviationTriggersRepath(t *testing.T) {
	grid := grids.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steering := board.NewSteeringSystem(board.NewPathFinder(grid), terrain, occupancy, 20)
	pusher := &pushOnce{grid: grid, size: 8}

	start := cellAtXY(grid, 0, 0)
	target := cellAtXY(grid, 4, 0)
	pushed := cellAtXY(grid, 3, 0)
	pusher.to = pushed

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var moveTo goke.Comp[board.MoveTo]
	var path goke.Comp[board.Path]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &moveTo, &path)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0] = world.Position{AABB: board.CellAABB(grid, start, 8)}
		moveTo.Slice(&f.Cursor)[0] = board.MoveTo{Target: target}
		occupancy.Enter(start, id)

		q = si.NewQueryBuilder(&cell, &path).Build()
	}})

	pusherHandle := ecs.RegSys(pusher)
	steerHandle := ecs.RegSys(steering)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(pusherHandle, d)
		ctx.Run(steerHandle, d)
		ctx.Sync()
	})

	ecs.Tick(time.Second) // computes the initial start->target path

	c, p := readCellAndPath(t, q, &cell, &path)
	if p.Length == 0 {
		t.Fatal("expected a path to have been computed on the first tick")
	}
	if c.ID != start {
		t.Fatalf("Cell.ID = %v, want %v (no movement yet, no deviation)", c.ID, start)
	}
	firstStep := p.Steps[0]
	if firstStep == pushed || firstStep == target {
		t.Fatalf("first path step %v should be an intermediate cell — grid too small for this test", firstStep)
	}

	pusher.armed = true
	ecs.Tick(time.Second) // pusher jumps the entity ahead; steering must notice and re-path

	c, p = readCellAndPath(t, q, &cell, &path)
	if c.ID != pushed {
		t.Errorf("Cell.ID = %v, want %v (deviation should resync bookkeeping to the actual cell)", c.ID, pushed)
	}
	if p.Length == 0 {
		t.Fatal("expected the path to have been recomputed after the deviation")
	}
	if p.Steps[p.Index] == firstStep {
		t.Error("recomputed path should route onward from the pushed-to cell, not resume the stale pre-deviation path")
	}
}

// TestSteeringSystem_Update_ArrivalStopsEntity guards against a real bug: on
// arrival the "arrived" branch removed MoveTo/Path but never zeroed
// Velocity — since the entity never matches this system's query again once
// MoveTo is gone, nothing ever stopped it, and it drifted off the board.
func TestSteeringSystem_Update_ArrivalStopsEntity(t *testing.T) {
	grid := grids.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steering := board.NewSteeringSystem(board.NewPathFinder(grid), terrain, occupancy, 20)

	start := cellAtXY(grid, 2, 0)
	target := start // already at the target — arrives on the very first tick

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var moveTo goke.Comp[board.MoveTo]
	var path goke.Comp[board.Path]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &moveTo, &path)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0] = world.Position{AABB: board.CellAABB(grid, start, 8)}
		vel.Slice(&f.Cursor)[0] = world.Velocity{Dir: geom.NewVec[float64](1, 0), Value: 50} // was already moving in
		moveTo.Slice(&f.Cursor)[0] = board.MoveTo{Target: target}
		occupancy.Enter(start, id)

		q = si.NewQueryBuilder(&cell, &vel).Build()
	}})

	steerHandle := ecs.RegSys(steering)
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

// TestSteeringSystem_Update_ArrivalSnapsToCellCenter guards that an entity
// arriving off-center (a natural consequence of continuous movement crossing
// a cell boundary at an arbitrary point) ends up exactly centered — needed
// for a board game, where units are expected to sit precisely on a cell.
func TestSteeringSystem_Update_ArrivalSnapsToCellCenter(t *testing.T) {
	grid := grids.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steering := board.NewSteeringSystem(board.NewPathFinder(grid), terrain, occupancy, 20)
	space := testSpace(t)
	steering.BindSpace(space)

	target := cellAtXY(grid, 2, 0)
	// Within arrivalEpsilon of the target cell's true center (25,5) for a 10px cell at column 2.
	offCenter := world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](20, 1), 8, 8)}

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var moveTo goke.Comp[board.MoveTo]
	var path goke.Comp[board.Path]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &moveTo, &path)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: target}
		pos.Slice(&f.Cursor)[0] = offCenter
		moveTo.Slice(&f.Cursor)[0] = board.MoveTo{Target: target}
		occupancy.Enter(target, id)
		space.Insert(id, offCenter.AABB)
		space.Flush(nil)

		q = si.NewQueryBuilder(&pos).Build()
	}})

	steerHandle := ecs.RegSys(steering)
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

// TestSteeringSystem_Update_ArrivalGlidesSmoothlyToCellCenter guards against
// a real bug: switching waypoints on cell-boundary crossing (rather than
// proximity to the cell's true center) made a large final correction happen
// in a single tick, visibly popping the entity into place. Arrival must
// glide in bounded steps and still land exactly on center.
func TestSteeringSystem_Update_ArrivalGlidesSmoothlyToCellCenter(t *testing.T) {
	grid := grids.NewSquareGrid(5, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	steering := board.NewSteeringSystem(board.NewPathFinder(grid), terrain, occupancy, 20)
	space := testSpace(t)
	steering.BindSpace(space)

	target := cellAtXY(grid, 2, 0)
	// Off the true center (25,5), within the target cell, well beyond arrivalEpsilon.
	offCenter := world.Position{AABB: plane.NewAABB(geom.NewVec[uint32](17, 1), 8, 8)}

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var vel goke.Comp[world.Velocity]
	var moveTo goke.Comp[board.MoveTo]
	var path goke.Comp[board.Path]
	var q *goke.Query

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &vel, &moveTo, &path)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: target}
		pos.Slice(&f.Cursor)[0] = offCenter
		moveTo.Slice(&f.Cursor)[0] = board.MoveTo{Target: target}
		occupancy.Enter(target, id)
		space.Insert(id, offCenter.AABB)
		space.Flush(nil)

		q = si.NewQueryBuilder(&pos).Build()
	}})

	steerHandle := ecs.RegSys(steering)
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

func readCellAndPath(t *testing.T, q *goke.Query, cell *goke.Comp[board.Cell], path *goke.Comp[board.Path]) (board.Cell, board.Path) {
	t.Helper()
	q.All()
	for q.Next() {
		cur := q.Cursor()
		cells := cell.Slice(cur)
		paths := path.Slice(cur)
		if len(cells) > 0 {
			return cells[0], paths[0]
		}
	}
	t.Fatal("expected to find the seeded entity")
	return board.Cell{}, board.Path{}
}
