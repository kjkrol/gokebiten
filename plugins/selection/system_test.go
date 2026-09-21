package selection

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/aabbworld/plane"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/camera"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/uid"
)

type pendingSeed struct {
	x, y, size float64
	id         *uid.UID64
}

// harness seeds entities, drives the system through a tick and reads Selected back;
// seed only queues, start performs the single Setup.
type harness struct {
	t         *testing.T
	space     *aabbworld.Space
	state     *Resources
	sys       *SelectionSystem
	handler   *DefaultEventHandler
	ecs       *goke.ECS
	pos       goke.Comp[world.Base]
	selectedQ *goke.Query
	handle    goke.Runnable
	pending   []pendingSeed
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	space, err := aabbworld.NewSpace(aabbworld.Config{
		Width: 1000, Height: 1000,
		BucketSize: 64, BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("aabbworld.NewSpace: %v", err)
	}

	cam := camera.NewFromSpace(1000, 1000, 0)

	state := &Resources{}
	sys := NewSelectionSystem(state, space, cam)
	handler := NewDefaultEventHandler(state)

	return &harness{t: t, space: space, state: state, sys: sys, handler: handler, ecs: goke.New()}
}

// seed queues a size x size entity at (x,y); the returned id is filled in by start.
func (h *harness) seed(x, y, size float64) *uid.UID64 {
	id := new(uid.UID64)
	h.pending = append(h.pending, pendingSeed{x: x, y: y, size: size, id: id})
	return id
}

// start builds the query and spawns every queued seed — call once, after all seed() calls.
func (h *harness) start() {
	h.t.Helper()
	h.ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		h.selectedQ = si.NewQueryBuilder().Include(goke.Include[Selected]()).Build()
		if len(h.pending) == 0 {
			return
		}
		f := si.NewFactory(&h.pos)
		f.Create(len(h.pending))
		i := 0
		for f.Next() {
			positions := h.pos.Slice(&f.Cursor)
			for j, id := range f.Cursor.IDs {
				spec := h.pending[i]
				*spec.id = id
				aabb := plane.NewAABB(geom.NewVec(spec.x, spec.y), spec.size, spec.size)
				positions[j].Pos = world.Position{AABB: aabb}
				h.space.Insert(id, &aabb)
				i++
			}
		}
	}})
	h.space.Flush(nil)

	h.handle = h.ecs.RegSys(h.sys)
	h.ecs.SetPlan(func(ctx goke.RunCtx, d time.Duration) {
		ctx.Run(h.handle, d)
		ctx.Sync()
	})
}

func (h *harness) click(x, y int, shift bool) {
	events := &control.InputEvents{}
	events.Modifiers.Shift = shift
	events.AddClickEvent(x, y, ebiten.MouseButtonLeft, control.ActionPress)
	events.AddClickEvent(x, y, ebiten.MouseButtonLeft, control.ActionRelease)
	h.handler.HandleEvents(events)
	h.ecs.Tick(time.Second)
}

func (h *harness) drag(x0, y0, x1, y1 int, shift bool) {
	events := &control.InputEvents{}
	events.Modifiers.Shift = shift
	events.AddClickEvent(x0, y0, ebiten.MouseButtonLeft, control.ActionPress)
	events.AddClickEvent(x1, y1, ebiten.MouseButtonLeft, control.ActionRelease)
	h.handler.HandleEvents(events)
	h.ecs.Tick(time.Second)
}

func (h *harness) isSelected(id uid.UID64) bool {
	h.t.Helper()
	h.selectedQ.All()
	for h.selectedQ.Next() {
		cur := h.selectedQ.Cursor()
		for _, got := range cur.IDs {
			if got == id {
				return true
			}
		}
	}
	return false
}

func TestSystem_Update_ClickSelectsHitEntity(t *testing.T) {
	h := newHarness(t)
	id := h.seed(50, 50, 10)
	h.start()

	h.click(55, 55, false)

	if !h.isSelected(*id) {
		t.Error("expected the entity under the click to be Selected")
	}
}

func TestSystem_Update_ClickOnEmptySpaceClearsSelection(t *testing.T) {
	h := newHarness(t)
	id := h.seed(50, 50, 10)
	h.start()

	h.click(55, 55, false)
	if !h.isSelected(*id) {
		t.Fatal("sanity check failed: expected entity to be selected after first click")
	}

	h.click(500, 500, false)

	if h.isSelected(*id) {
		t.Error("expected a non-additive click on empty space to clear the previous selection")
	}
}

func TestSystem_Update_DragSelectsEntitiesInsideBox_ReplacesOutside(t *testing.T) {
	h := newHarness(t)
	inside1 := h.seed(20, 20, 10)
	inside2 := h.seed(80, 80, 10)
	outside := h.seed(500, 500, 10)
	h.start()

	h.click(505, 505, false)
	if !h.isSelected(*outside) {
		t.Fatal("sanity check failed: expected outside entity to be selected first")
	}

	h.drag(0, 0, 100, 100, false)

	if !h.isSelected(*inside1) || !h.isSelected(*inside2) {
		t.Errorf("expected both entities inside the drag box to be Selected")
	}
	if h.isSelected(*outside) {
		t.Error("expected the entity outside the drag box to lose Selected (non-additive drag replaces)")
	}
}

func TestSystem_Update_ShiftClickAddsToExistingSelection(t *testing.T) {
	h := newHarness(t)
	first := h.seed(20, 20, 10)
	second := h.seed(200, 200, 10)
	h.start()

	h.click(25, 25, false)
	if !h.isSelected(*first) {
		t.Fatal("sanity check failed: expected first entity to be selected")
	}

	h.click(205, 205, true)

	if !h.isSelected(*first) {
		t.Error("expected the first selection to survive a Shift-click elsewhere")
	}
	if !h.isSelected(*second) {
		t.Error("expected the Shift-clicked entity to also be Selected")
	}
}

func TestSystem_Update_DragAcrossMultipleTicks(t *testing.T) {
	h := newHarness(t)
	id := h.seed(50, 50, 10)
	h.start()

	press := &control.InputEvents{}
	press.AddClickEvent(40, 40, ebiten.MouseButtonLeft, control.ActionPress)
	h.handler.HandleEvents(press)
	h.ecs.Tick(time.Second)

	if h.isSelected(*id) {
		t.Fatal("sanity check failed: press alone (no release yet) should not select anything")
	}

	release := &control.InputEvents{}
	release.AddClickEvent(60, 60, ebiten.MouseButtonLeft, control.ActionRelease)
	h.handler.HandleEvents(release)
	h.ecs.Tick(time.Second)

	if !h.isSelected(*id) {
		t.Error("expected the entity inside the drag box to be Selected after release, even though press/release arrived in separate HandleEvents calls")
	}
}

func TestSystem_Update_SelectByID_TagsExactlyGivenEntities(t *testing.T) {
	h := newHarness(t)
	target := h.seed(20, 20, 10)
	other := h.seed(200, 200, 10)
	h.start()

	h.click(205, 205, false)
	if !h.isSelected(*other) {
		t.Fatal("sanity check failed: expected other to be selected first")
	}

	h.state.PendingIDs = []uid.UID64{*target}
	h.ecs.Tick(time.Second)

	if !h.isSelected(*target) {
		t.Error("expected Select to tag the given entity as Selected")
	}
	if h.isSelected(*other) {
		t.Error("expected Select to replace the previous selection, not add to it")
	}
}

func TestSystem_DragBox_TracksLiveDragState(t *testing.T) {
	h := newHarness(t)
	h.start()

	if _, _, dragging := h.state.DragBox(); dragging {
		t.Fatal("sanity check failed: expected no drag in progress before any input")
	}

	press := &control.InputEvents{}
	press.AddClickEvent(10, 10, ebiten.MouseButtonLeft, control.ActionPress)
	h.handler.HandleEvents(press)

	start, current, dragging := h.state.DragBox()
	if !dragging {
		t.Fatal("expected dragging=true right after a press")
	}
	if start != geom.NewVec(10, 10) || current != geom.NewVec(10, 10) {
		t.Errorf("start/current = %v/%v, want (10,10)/(10,10)", start, current)
	}

	move := &control.InputEvents{MousePos: geom.NewVec(40, 60)}
	h.handler.HandleEvents(move)

	start, current, dragging = h.state.DragBox()
	if !dragging {
		t.Error("expected dragging to remain true while the button is still held")
	}
	if start != geom.NewVec(10, 10) {
		t.Errorf("start = %v, want unchanged (10,10)", start)
	}
	if current != geom.NewVec(40, 60) {
		t.Errorf("current = %v, want (40,60) (updated from MousePos with no click event)", current)
	}

	release := &control.InputEvents{}
	release.AddClickEvent(40, 60, ebiten.MouseButtonLeft, control.ActionRelease)
	h.handler.HandleEvents(release)

	if _, _, dragging := h.state.DragBox(); dragging {
		t.Error("expected dragging=false after release")
	}
}

func TestSelectionSystem_WorldBox_SelectsOnBothSidesOfTheSeam(t *testing.T) {
	space, err := aabbworld.NewSpace(aabbworld.Config{
		Width: 1000, Height: 1000, Edges: aabbworld.Torus,
		BucketSize: 64, BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("aabbworld.NewSpace: %v", err)
	}
	before, after, elsewhere := uid.UID64(1), uid.UID64(2), uid.UID64(3)
	space.Insert(before, ptr(plane.NewAABB(geom.NewVec(960, 502), 5, 5)))
	space.Insert(after, ptr(plane.NewAABB(geom.NewVec(20, 502), 5, 5)))
	space.Insert(elsewhere, ptr(plane.NewAABB(geom.NewVec(500, 502), 5, 5)))
	space.Flush(nil)

	cam := camera.NewFromSpaceWithConfig(1000, 1000, aabbworld.Torus, camera.Config{ViewportWidth: 200, ViewportHeight: 200})
	cam.MoveTo(950, 500)
	sys := &SelectionSystem{camera: cam, space: space}

	box := sys.worldBox(geom.NewVec(0, 0), geom.NewVec(200, 10))
	if w := box.BottomRight.X - box.TopLeft.X; w > 200 {
		t.Errorf("worldBox is %v wide, want the 200 that was dragged", w)
	}

	hit := map[uid.UID64]bool{}
	space.Query(box, aabbworld.AnyCapability, func(id uid.UID64) { hit[id] = true })
	if !hit[before] || !hit[after] || hit[elsewhere] || len(hit) != 2 {
		t.Errorf("drag across the seam hit %v, want the entities either side of it and nothing else", hit)
	}
}
