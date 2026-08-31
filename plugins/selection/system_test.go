package selection

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/control"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokebiten/render"
	"github.com/kjkrol/gokg"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/gokg/spatial"
	"github.com/kjkrol/uid"
)

type pendingSeed struct {
	x, y, size uint32
	id         *uid.UID64
}

// harness bundles everything a test needs to seed entities, drive System
// through a tick, and read back Selected. goke allows exactly one Setup
// call per ECS, so seed() only queues specs — start() performs the single
// Setup that builds the query and spawns everything queued.
type harness struct {
	t         *testing.T
	space     *gokg.Space
	state     *State
	sys       *System
	handler   *DefaultEventHandler
	ecs       *goke.ECS
	pos       goke.Comp[world.Position]
	selectedQ *goke.Query
	handle    goke.Runnable
	pending   []pendingSeed
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	space, err := gokg.NewSpace(gokg.Config{
		Width: 1000, Height: 1000,
		BucketSize: spatial.ResolutionFrom(64), BucketCapacity: 16, OpsBufferSize: 64,
	})
	if err != nil {
		t.Fatalf("gokg.NewSpace: %v", err)
	}

	surface := plane.NewEuclidean2D[uint32](1000, 1000)
	camera := render.NewBasicCamera(surface, geom.NewAABBAt(geom.NewVec[uint32](0, 0), 1000, 1000))

	state := &State{}
	sys := NewSystem(state)
	sys.bindSpace(space)
	sys.bindCamera(camera)
	handler := NewDefaultEventHandler(state)

	return &harness{t: t, space: space, state: state, sys: sys, handler: handler, ecs: goke.New()}
}

// seed queues an entity at world position (x,y) sized size x size — the
// returned pointer is filled in once start() runs.
func (h *harness) seed(x, y, size uint32) *uid.UID64 {
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
				positions[j] = world.Position{AABB: aabb}
				h.space.Insert(id, aabb)
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

	// pre-select outside so we can verify a non-additive drag drops it.
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

	h.click(205, 205, true) // shift held: additive

	if !h.isSelected(*first) {
		t.Error("expected the first selection to survive a Shift-click elsewhere")
	}
	if !h.isSelected(*second) {
		t.Error("expected the Shift-clicked entity to also be Selected")
	}
}

// TestSystem_Update_DragAcrossMultipleTicks guards against a real usage
// difference from the other tests here: a real mouse drag delivers Press
// and Release in SEPARATE Game.Update calls (separate HandleEvents calls,
// with a tick in between), not batched into one InputEvents like click()/drag() do.
func TestSystem_Update_DragAcrossMultipleTicks(t *testing.T) {
	h := newHarness(t)
	id := h.seed(50, 50, 10)
	h.start()

	press := &control.InputEvents{}
	press.AddClickEvent(40, 40, ebiten.MouseButtonLeft, control.ActionPress)
	h.handler.HandleEvents(press)
	h.ecs.Tick(time.Second) // nothing pending yet — press alone shouldn't select

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

	h.click(205, 205, false) // pre-select other, to confirm Select() replaces it
	if !h.isSelected(*other) {
		t.Fatal("sanity check failed: expected other to be selected first")
	}

	h.sys.Select([]uid.UID64{*target})
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
	if start != geom.NewVec[int32](10, 10) || current != geom.NewVec[int32](10, 10) {
		t.Errorf("start/current = %v/%v, want (10,10)/(10,10)", start, current)
	}

	move := &control.InputEvents{MousePos: geom.NewVec[int32](40, 60)}
	h.handler.HandleEvents(move)

	start, current, dragging = h.state.DragBox()
	if !dragging {
		t.Error("expected dragging to remain true while the button is still held")
	}
	if start != geom.NewVec[int32](10, 10) {
		t.Errorf("start = %v, want unchanged (10,10)", start)
	}
	if current != geom.NewVec[int32](40, 60) {
		t.Errorf("current = %v, want (40,60) (updated from MousePos with no click event)", current)
	}

	release := &control.InputEvents{}
	release.AddClickEvent(40, 60, ebiten.MouseButtonLeft, control.ActionRelease)
	h.handler.HandleEvents(release)

	if _, _, dragging := h.state.DragBox(); dragging {
		t.Error("expected dragging=false after release")
	}
}
