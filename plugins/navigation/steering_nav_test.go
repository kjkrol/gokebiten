package navigation

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/plugins/board"
	"github.com/kjkrol/gram/plugins/world"
	"github.com/kjkrol/uid"
)

// profiledWorld is one navigated entity with a given Steering profile on an open square grid,
// ticked at 60 TPS through navigation, steering and movement.
type profiledWorld struct {
	grid  board.Grid
	ecs   *goke.ECS
	id    uid.UID64
	pos   goke.Comp[world.Base]
	order goke.OptComp[MoveOrder]
	q     *goke.Query
}

func newProfiledWorld(t *testing.T, w, h uint32, start, target board.CellID, profile world.Steering, withSteering bool) *profiledWorld {
	t.Helper()
	pw := &profiledWorld{grid: board.DefaultGrids{}.Square(w, h, legCellSize)}
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	nav := newNavigationSystem(newPathFinder(pw.grid, terrain, occupancy), pw.grid, terrain, occupancy)
	space := testSpace(t)
	nav.BindSpace(space)

	pw.ecs = goke.New()
	pw.ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		var cell goke.Comp[board.Cell]
		var pos goke.Comp[world.Base]
		var order goke.Comp[MoveOrder]
		var steer goke.Comp[world.Steering]
		comps := []goke.Addable{&cell, &pos, &order}
		if withSteering {
			comps = append(comps, &steer)
		}
		f := si.NewFactory(comps...)
		f.Create(1)
		f.Next()
		pw.id = f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: start}
		pos.Slice(&f.Cursor)[0].Pos = world.Position{AABB: board.CellAABB(pw.grid, start, legEntitySize)}
		order.Slice(&f.Cursor)[0] = MoveOrder{Target: target}
		if withSteering {
			steer.Slice(&f.Cursor)[0] = profile
		}
		occupancy.Enter(start, pw.id)
		pw.q = si.NewQueryBuilder(&pw.pos).Optional(&pw.order).Build()
	}})

	navHandle := pw.ecs.RegSys(nav)
	steeringHandle := pw.ecs.RegSys(world.NewSteeringSystem())
	moveHandle := pw.ecs.RegSys(world.NewMoveSystem(space))
	pw.ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(navHandle, d)
		ctx.Run(steeringHandle, d)
		ctx.Run(moveHandle, d)
		ctx.Sync()
	})
	return pw
}

// tick advances one tick and reports the entity's velocity, centre and whether it still has an order.
func (pw *profiledWorld) tick() (vel world.Velocity, centre geom.Vec, ordered bool) {
	pw.ecs.Tick(time.Second / 60)
	pw.q.All()
	for pw.q.Next() {
		cur := pw.q.Cursor()
		b := pw.pos.Slice(cur)[0]
		return b.Vel, board.Center(b.Pos), pw.order.Slice(cur) != nil
	}
	return
}

func (pw *profiledWorld) cellAt(x, y uint32) board.CellID {
	c, _ := pw.grid.CellIndex(x, y)
	return c
}

func TestNavigation_TurnsBeforeTheBendAndNeverStops(t *testing.T) {
	const turnRate = 0.1
	pw := newProfiledWorld(t, 6, 6, board.CellID(0), board.CellID(0), world.Steering{}, true)
	pw = newProfiledWorld(t, 6, 6, pw.cellAt(0, 2), pw.cellAt(5, 4), world.Steering{MaxSpeed: 64, TurnRate: turnRate}, true)

	var prev float64
	haveHeading := false
	turned := 0.0
	for tick := range 60 * 20 {
		vel, _, ordered := pw.tick()
		if !ordered {
			if turned < 0.2 {
				t.Errorf("arrived having turned only %.2f rad in total; the route should have bent", turned)
			}
			return
		}
		if vel.Value <= 0 {
			t.Fatalf("tick %d: Velocity.Value = %v on the way; a unit must never stop between waypoints", tick, vel.Value)
		}
		h := math.Atan2(vel.Dir.Y, vel.Dir.X)
		if haveHeading {
			delta := math.Abs(math.Mod(h-prev+3*math.Pi, 2*math.Pi) - math.Pi)
			if delta > turnRate+1e-9 {
				t.Fatalf("tick %d: heading swung by %.3f rad in one tick, want at most TurnRate %.3f", tick, delta, turnRate)
			}
			turned += delta
		}
		prev, haveHeading = h, true
	}
	t.Fatal("never arrived")
}

func TestNavigation_PassesAWaypointByProjectionNotDistance(t *testing.T) {
	grid := board.DefaultGrids{}.Square(5, 1, 10)
	terrain := board.NewTerrainMap()
	terrain.SetAll(board.CellKind{Cost: 1, Passable: true})
	occupancy := &board.SingleOccupancy{}
	nav := newNavigationSystem(newPathFinder(grid, terrain, occupancy), grid, terrain, occupancy)
	at := func(x uint32) board.CellID { c, _ := grid.CellIndex(x, 0); return c }

	var cell goke.Comp[board.Cell]
	var pos goke.Comp[world.Base]
	var order goke.Comp[MoveOrder]
	var profile goke.Comp[world.Steering]
	var q *goke.Query
	ecs := goke.New()
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&cell, &pos, &order, &profile)
		f.Create(1)
		f.Next()
		id := f.Cursor.IDs[0]
		cell.Slice(&f.Cursor)[0] = board.Cell{ID: at(1)}
		// past the plane through cell 2's centre (x = 25), yet 3.2 units from that centre
		pos.Slice(&f.Cursor)[0].Pos = world.Position{AABB: geomBox(26, 8, 4)}
		var mt MoveOrder
		mt.Target = at(3)
		mt.Path.Steps[0], mt.Path.Steps[1] = at(2), at(3)
		mt.Path.Length = 2
		mt.Leg = Leg{From: at(1), To: at(2), Active: true}
		order.Slice(&f.Cursor)[0] = mt
		profile.Slice(&f.Cursor)[0] = world.Steering{MaxSpeed: 20}
		for _, c := range mt.Leg.cells() {
			occupancy.Enter(c, id)
		}
		q = si.NewQueryBuilder(&cell, &order).Build()
	}})
	navHandle := ecs.RegSys(nav)
	ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) { ctx.Run(navHandle, d); ctx.Sync() })

	ecs.Tick(time.Second / 60)

	c, mt := readCellAndMoveOrder(t, q, &cell, &order)
	if mt.Path.Index != 1 {
		t.Errorf("Path.Index = %d, want 1: the waypoint behind the plane counts as passed", mt.Path.Index)
	}
	if mt.Leg.Active {
		t.Error("the leg onto the passed waypoint is still active")
	}
	if c.ID != at(2) {
		t.Errorf("Cell = %v, want %v", c.ID, at(2))
	}
}

func TestNavigation_BrakesToRestOnTheGoal(t *testing.T) {
	pw := newProfiledWorld(t, 6, 1, board.CellID(0), board.CellID(0), world.Steering{}, true)
	target := pw.cellAt(5, 0)
	pw = newProfiledWorld(t, 6, 1, pw.cellAt(0, 0), target, world.Steering{MaxSpeed: 64, Accel: 128, V0: 16}, true)

	var speeds []float64
	for range 60 * 20 {
		vel, centre, ordered := pw.tick()
		if ordered {
			if vel.Value <= 0 {
				t.Fatalf("Velocity.Value = %v before arriving; braking must not stop the unit short", vel.Value)
			}
			speeds = append(speeds, vel.Value)
			continue
		}
		if vel.Value != 0 {
			t.Errorf("Velocity.Value = %v after arriving, want 0", vel.Value)
		}
		if want := pw.grid.CellCenter(target); centre != want {
			t.Errorf("came to rest at %v, want the goal's centre %v", centre, want)
		}
		n := len(speeds)
		if n < 3 || speeds[n-1] >= speeds[n-3] {
			t.Errorf("last speeds %v: want them falling as the goal comes up", speeds[max(0, n-4):])
		}
		if peak := maxOf(speeds); peak != 64 {
			t.Errorf("peak speed %v, want the profile's 64 on the way", peak)
		}
		return
	}
	t.Fatal("never arrived")
}

func TestNavigation_LeavesAloneAnEntityWithoutSteering(t *testing.T) {
	pw := newProfiledWorld(t, 5, 1, board.CellID(0), board.CellID(0), world.Steering{}, true)
	start, target := pw.cellAt(0, 0), pw.cellAt(4, 0)
	pw = newProfiledWorld(t, 5, 1, start, target, world.Steering{}, false)

	before := pw.grid.CellCenter(start)
	for range 30 {
		vel, centre, ordered := pw.tick()
		if !ordered || centre != before || vel.Value != 0 {
			t.Fatalf("an entity without Steering was navigated: ordered=%v centre=%v vel=%v", ordered, centre, vel)
		}
	}
}

func TestLookahead(t *testing.T) {
	have := geom.NewVec(0, 0)
	route := []geom.Vec{geom.NewVec(10, 0), geom.NewVec(10, 10)}
	for name, tc := range map[string]struct {
		reach float64
		want  geom.Vec
	}{
		"no reach aims at the first waypoint":     {0, geom.NewVec(10, 0)},
		"within the first segment":                {4, geom.NewVec(4, 0)},
		"past the bend":                           {14, geom.NewVec(10, 4)},
		"beyond the route ends at its last point": {40, geom.NewVec(10, 10)},
	} {
		if got := lookahead(have, route, tc.reach); got != tc.want {
			t.Errorf("%s: lookahead(reach %v) = %v, want %v", name, tc.reach, got, tc.want)
		}
	}
	if got := lookahead(have, nil, 5); got != have {
		t.Errorf("an empty route gave %v, want the position itself", got)
	}
}

func geomBox(cx, cy, size float64) plane.AABB {
	return plane.NewAABB(geom.NewVec(cx-size/2, cy-size/2), size, size)
}

func maxOf(xs []float64) float64 {
	m := math.Inf(-1)
	for _, x := range xs {
		m = max(m, x)
	}
	return m
}
