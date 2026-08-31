package board_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/board"
	"github.com/kjkrol/gokebiten/plugins/board/grids"
	"github.com/kjkrol/gokebiten/plugins/selection"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

func TestCommandSystem_Update_RetargetsOnlySelectedEntities(t *testing.T) {
	grid := grids.NewSquareGrid(10, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}

	start := cellAtXY(grid, 0, 0)
	oldTarget := cellAtXY(grid, 3, 0)
	newTarget := cellAtXY(grid, 8, 0)

	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	camera := render.NewBasicCamera(surface, geom.NewAABBAt(geom.NewVec[uint32](0, 0), 1000, 1000))

	cmdState := &board.CommandState{}
	cmds := board.NewCommandSystem(board.NewPathFinder(grid), terrain, occupancy, cmdState)
	cmdHandler := board.NewDefaultCommandEventHandler(grid, camera, cmdState)
	selState := &selection.State{}
	selSys := selection.NewSystem(selState)

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var moveTo goke.Comp[board.MoveTo]
	var path goke.Comp[board.Path]
	var readQuery *goke.Query
	var selectedID, otherID uid.UID64

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &moveTo, &path)
		f.Create(2)
		f.Next()
		ids := f.Cursor.IDs
		selectedID, otherID = ids[0], ids[1]

		cells := cell.Slice(&f.Cursor)
		positions := pos.Slice(&f.Cursor)
		moveTos := moveTo.Slice(&f.Cursor)
		paths := path.Slice(&f.Cursor)
		for i := range ids {
			cells[i] = board.Cell{ID: start}
			positions[i] = world.Position{AABB: board.CellAABB(grid, start, 8)}
			moveTos[i] = board.MoveTo{Target: oldTarget}
			paths[i] = board.Path{Length: 1}
		}

		readQuery = si.NewQueryBuilder(&moveTo, &path).Build()
		selSys.Init(si)
		cmds.Init(si)
	}})

	selHandle := ecs.RegSys(selSys)
	cmdHandle := ecs.RegSys(cmds)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(selHandle, d)
		ctx.Run(cmdHandle, d)
		ctx.Sync()
	})

	selSys.Select([]uid.UID64{selectedID})
	ecs.Tick(time.Second)

	center := grid.CellCenter(newTarget)
	sx, sy := camera.ToScreen(float32(center.X), float32(center.Y))
	events := &control.InputEvents{}
	events.AddClickEvent(int(sx), int(sy), ebiten.MouseButtonRight, control.ActionPress)
	cmdHandler.HandleEvents(events)
	ecs.Tick(time.Second)

	got := map[uid.UID64]struct {
		moveTo board.MoveTo
		path   board.Path
	}{}
	readQuery.All()
	for readQuery.Next() {
		cur := readQuery.Cursor()
		moveTos := moveTo.Slice(cur)
		paths := path.Slice(cur)
		for i, id := range cur.IDs {
			got[id] = struct {
				moveTo board.MoveTo
				path   board.Path
			}{moveTos[i], paths[i]}
		}
	}

	if got[selectedID].moveTo.Target != newTarget {
		t.Errorf("selected entity's MoveTo.Target = %v, want %v", got[selectedID].moveTo.Target, newTarget)
	}
	if got[selectedID].path.Length != 8 {
		t.Errorf("selected entity's Path.Length = %d, want 8 (freshly computed by CommandSystem, straight 8-cell line to newTarget)", got[selectedID].path.Length)
	} else if got[selectedID].path.Steps[7] != newTarget {
		t.Errorf("selected entity's last path step = %v, want newTarget %v", got[selectedID].path.Steps[7], newTarget)
	}

	if got[otherID].moveTo.Target != oldTarget {
		t.Errorf("unselected entity's MoveTo.Target = %v, want unchanged %v", got[otherID].moveTo.Target, oldTarget)
	}
	if got[otherID].path.Length != 1 {
		t.Errorf("unselected entity's Path.Length = %d, want unchanged 1", got[otherID].path.Length)
	}
}

// TestCommandSystem_Update_AssignsFreshOrderToIdleSelectedEntity guards the
// case SteeringSystem produces once a unit arrives: it removes MoveTo/Path
// entirely, so a right-click must still be able to give that idle, Selected
// entity a brand new order — not just retarget units already in transit.
func TestCommandSystem_Update_AssignsFreshOrderToIdleSelectedEntity(t *testing.T) {
	grid := grids.NewSquareGrid(10, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}

	start := cellAtXY(grid, 0, 0)
	newTarget := cellAtXY(grid, 8, 0)

	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	camera := render.NewBasicCamera(surface, geom.NewAABBAt(geom.NewVec[uint32](0, 0), 1000, 1000))

	cmdState := &board.CommandState{}
	cmds := board.NewCommandSystem(board.NewPathFinder(grid), terrain, occupancy, cmdState)
	cmdHandler := board.NewDefaultCommandEventHandler(grid, camera, cmdState)
	selState := &selection.State{}
	selSys := selection.NewSystem(selState)

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var moveTo goke.OptComp[board.MoveTo]
	var path goke.OptComp[board.Path]
	var readQuery *goke.Query
	var idleID uid.UID64

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos)
		f.Create(1)
		f.Next()
		idleID = f.Cursor.IDs[0]

		cells := cell.Slice(&f.Cursor)
		positions := pos.Slice(&f.Cursor)
		cells[0] = board.Cell{ID: start}
		positions[0] = world.Position{AABB: board.CellAABB(grid, start, 8)}

		readQuery = si.NewQueryBuilder().Optional(&moveTo, &path).Build()
		selSys.Init(si)
		cmds.Init(si)
	}})

	selHandle := ecs.RegSys(selSys)
	cmdHandle := ecs.RegSys(cmds)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(selHandle, d)
		ctx.Run(cmdHandle, d)
		ctx.Sync()
	})

	selSys.Select([]uid.UID64{idleID})
	ecs.Tick(time.Second)

	readQuery.All()
	for readQuery.Next() {
		cur := readQuery.Cursor()
		if moveTo.Present(cur) {
			t.Fatal("sanity check failed: expected the idle entity to start without MoveTo")
		}
	}

	center := grid.CellCenter(newTarget)
	sx, sy := camera.ToScreen(float32(center.X), float32(center.Y))
	events := &control.InputEvents{}
	events.AddClickEvent(int(sx), int(sy), ebiten.MouseButtonRight, control.ActionPress)
	cmdHandler.HandleEvents(events)
	ecs.Tick(time.Second)

	var gotMoveTo board.MoveTo
	var gotPath board.Path
	found := false
	readQuery.All()
	for readQuery.Next() {
		cur := readQuery.Cursor()
		if !moveTo.Present(cur) {
			continue
		}
		moveTos := moveTo.Slice(cur)
		paths := path.Slice(cur)
		for i, id := range cur.IDs {
			if id == idleID {
				gotMoveTo, gotPath, found = moveTos[i], paths[i], true
			}
		}
	}

	if !found {
		t.Fatal("expected the idle entity to have MoveTo/Path after a right-click order")
	}
	if gotMoveTo.Target != newTarget {
		t.Errorf("idle entity's MoveTo.Target = %v, want %v", gotMoveTo.Target, newTarget)
	}
	if gotPath.Length != 8 {
		t.Errorf("idle entity's Path.Length = %d, want 8 (freshly computed by CommandSystem, straight 8-cell line to newTarget)", gotPath.Length)
	} else if gotPath.Steps[7] != newTarget {
		t.Errorf("idle entity's last path step = %v, want newTarget %v", gotPath.Steps[7], newTarget)
	}
}

// TestCommandSystem_Update_UnreachableTargetLeavesInFlightEntityUntouched
// covers the RTS convention this system exists for: ordering a moving unit
// to a wall (or an occupied cell) must not interrupt its current route —
// the order is silently dropped, not queued with a corrupted/wrong target.
func TestCommandSystem_Update_UnreachableTargetLeavesInFlightEntityUntouched(t *testing.T) {
	grid := grids.NewSquareGrid(10, 1, 10)
	terrain := board.NewTerrainMap(board.CellProps{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}

	start := cellAtXY(grid, 0, 0)
	oldTarget := cellAtXY(grid, 3, 0)
	wall := cellAtXY(grid, 8, 0)
	terrain.Set(wall, board.CellProps{Cost: 1, Passable: false})

	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	camera := render.NewBasicCamera(surface, geom.NewAABBAt(geom.NewVec[uint32](0, 0), 1000, 1000))

	cmdState := &board.CommandState{}
	cmds := board.NewCommandSystem(board.NewPathFinder(grid), terrain, occupancy, cmdState)
	cmdHandler := board.NewDefaultCommandEventHandler(grid, camera, cmdState)
	selState := &selection.State{}
	selSys := selection.NewSystem(selState)

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Position]
	var moveTo goke.Comp[board.MoveTo]
	var path goke.Comp[board.Path]
	var readQuery *goke.Query
	var movingID uid.UID64

	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &moveTo, &path)
		f.Create(1)
		f.Next()
		movingID = f.Cursor.IDs[0]

		cells := cell.Slice(&f.Cursor)
		positions := pos.Slice(&f.Cursor)
		moveTos := moveTo.Slice(&f.Cursor)
		paths := path.Slice(&f.Cursor)
		cells[0] = board.Cell{ID: start}
		positions[0] = world.Position{AABB: board.CellAABB(grid, start, 8)}
		moveTos[0] = board.MoveTo{Target: oldTarget}
		paths[0] = board.Path{Length: 1}

		readQuery = si.NewQueryBuilder(&moveTo, &path).Build()
		selSys.Init(si)
		cmds.Init(si)
	}})

	selHandle := ecs.RegSys(selSys)
	cmdHandle := ecs.RegSys(cmds)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(selHandle, d)
		ctx.Run(cmdHandle, d)
		ctx.Sync()
	})

	selSys.Select([]uid.UID64{movingID})
	ecs.Tick(time.Second)

	center := grid.CellCenter(wall)
	sx, sy := camera.ToScreen(float32(center.X), float32(center.Y))
	events := &control.InputEvents{}
	events.AddClickEvent(int(sx), int(sy), ebiten.MouseButtonRight, control.ActionPress)
	cmdHandler.HandleEvents(events)
	ecs.Tick(time.Second)

	readQuery.All()
	found := false
	for readQuery.Next() {
		cur := readQuery.Cursor()
		moveTos := moveTo.Slice(cur)
		paths := path.Slice(cur)
		for i, id := range cur.IDs {
			if id != movingID {
				continue
			}
			found = true
			if moveTos[i].Target != oldTarget {
				t.Errorf("MoveTo.Target = %v, want unchanged %v (order to a wall must be dropped)", moveTos[i].Target, oldTarget)
			}
			if paths[i].Length != 1 {
				t.Errorf("Path.Length = %d, want unchanged 1 (order to a wall must be dropped)", paths[i].Length)
			}
		}
	}
	if !found {
		t.Fatal("expected the moving entity to still have MoveTo/Path")
	}
}
