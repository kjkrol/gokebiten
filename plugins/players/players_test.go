package players_test

import (
	"errors"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gram/camera"
	"github.com/kjkrol/gram/control"
	"github.com/kjkrol/gram/plugin"
	"github.com/kjkrol/gram/plugins/players"
	"github.com/kjkrol/gram/plugins/world"
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

// order and note are two command types of a made-up plugin.
type order struct{ Cell int }
type note struct{ Text string }

// rig is a players plugin over a 1000×1000 world with one local player and an order inbox.
type rig struct {
	t      *testing.T
	w      *world.Plugin
	p      *players.Plugin
	local  *players.Player
	orders *players.Inbox[order]
}

func newRig(t *testing.T, cfg ...camera.Config) *rig {
	t.Helper()
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	if len(cfg) > 0 {
		w.Res.Camera = camera.NewFromSpaceWithConfig(1000, 1000, 0, cfg[0])
	}
	p := players.NewPlugin(w)
	return &rig{t: t, w: w, p: p, local: p.Local("tester"), orders: p.Listen[order]()}
}

// start installs the plugin into an ECS whose plan is players' RunPlan, so camera commands land.
func (r *rig) start() *goke.ECS {
	r.t.Helper()
	ctx := &installCtx{ecs: goke.New()}
	if err := r.p.Install(ctx); err != nil {
		r.t.Fatal(err)
	}
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)
	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) { r.p.RunPlan(rc, d) })
	return ctx.ecs
}

func (r *rig) bind(b ...players.Binding) {
	r.t.Helper()
	if err := r.local.Bind(b...); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) handle(ev *control.InputEvents) { r.p.EventHandler().HandleEvents(ev) }

func (r *rig) drained() []players.Issued[order] {
	var got []players.Issued[order]
	r.orders.Drain(func(i players.Issued[order]) { got = append(got, i) })
	return got
}

func orderOf(n int) func(players.Context) (order, bool) {
	return func(players.Context) (order, bool) { return order{n}, true }
}

func TestBind_RefusesTwoBindingsOnOneTrigger(t *testing.T) {
	r := newRig(t)
	r.bind(players.Command(players.KeyPress{Key: ebiten.KeyA}, "one", orderOf(1)))
	err := r.local.Bind(players.Command(players.KeyPress{Key: ebiten.KeyA}, "two", orderOf(2)))
	if err == nil {
		t.Fatal("two bindings on KeyPress A were accepted")
	}
	if err := r.local.Bind(players.Command(players.KeyPress{Key: ebiten.KeyA, Mods: players.Mods{Shift: true}}, "shifted", orderOf(3))); err != nil {
		t.Errorf("Shift+A beside A: %v, want accepted as a different trigger", err)
	}
	if err := r.local.Bind(players.Binding{Label: "bare"}); err == nil {
		t.Error("a Binding not built with Command was accepted")
	}
}

func TestIssue_RefusesACommandNobodyListensFor(t *testing.T) {
	r := newRig(t)
	if err := r.p.Issue(r.local, note{"hi"}); !errors.Is(err, players.ErrUnknownCommand) {
		t.Errorf("Issue(note) = %v, want ErrUnknownCommand", err)
	}
	if err := r.p.Issue(nil, order{7}); err != nil {
		t.Fatalf("Issue(order) = %v", err)
	}
	if got := r.drained(); len(got) != 1 || got[0].Command.Cell != 7 || got[0].Player != nil {
		t.Errorf("drained %v, want order 7 from nobody", got)
	}
	if !r.orders.Empty() {
		t.Error("the inbox is not empty after Drain")
	}
}

func TestListen_TwiceForOneTypePanics(t *testing.T) {
	r := newRig(t)
	defer func() {
		if recover() == nil {
			t.Error("a second Listen[order] did not panic")
		}
	}()
	r.p.Listen[order]()
}

func TestSetup_PanicsOnABindingNobodyListensFor(t *testing.T) {
	r := newRig(t)
	r.bind(players.Command(players.KeyPress{Key: ebiten.KeyN}, "unheard", func(players.Context) (note, bool) { return note{}, true }))
	ctx := &installCtx{ecs: goke.New()}
	if err := r.p.Install(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Error("Setup accepted a binding whose command nobody listens for")
		}
	}()
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	ctx.ecs.Setup(systems...)
}

func TestKeysAndButtons_FireWithExactlyTheirModifiers(t *testing.T) {
	r := newRig(t)
	r.bind(
		players.Command(players.KeyPress{Key: ebiten.KeyA}, "a", orderOf(1)),
		players.Command(players.KeyPress{Key: ebiten.KeyA, Mods: players.Mods{Shift: true}}, "shift a", orderOf(2)),
		players.Command(players.ButtonPress{Button: ebiten.MouseButtonRight}, "right", func(c players.Context) (order, bool) {
			return order{int(c.Cursor.X)}, true
		}),
	)
	ev := &control.InputEvents{}
	ev.AddKeyEvent(ebiten.KeyA, control.ActionPress)
	ev.AddKeyEvent(ebiten.KeyA, control.ActionRelease)
	ev.AddClickEvent(40, 5, ebiten.MouseButtonRight, control.ActionPress)
	r.handle(ev)
	got := r.drained()
	if len(got) != 2 || got[0].Command.Cell != 1 || got[1].Command.Cell != 40 || got[0].Player != r.local {
		t.Errorf("plain A and a right click at 40 issued %v, want orders 1 and 40 from the local player", got)
	}

	ev = &control.InputEvents{}
	ev.Modifiers.Shift = true
	ev.AddKeyEvent(ebiten.KeyA, control.ActionPress)
	r.handle(ev)
	if got := r.drained(); len(got) != 1 || got[0].Command.Cell != 2 {
		t.Errorf("Shift+A issued %v, want order 2 alone", got)
	}
}

func TestDrag_FiresOnReleaseAndShowsWhileHeld(t *testing.T) {
	r := newRig(t)
	r.bind(players.Command(players.Drag{Button: ebiten.MouseButtonLeft}, "box", func(c players.Context) (order, bool) {
		return order{int(c.Start.X)*1000 + int(c.Cursor.X)}, true
	}))
	if _, _, dragging := r.local.DragBox(); dragging {
		t.Fatal("dragging before any input")
	}

	press := &control.InputEvents{MousePos: geom.NewVec(10, 10)}
	press.AddClickEvent(10, 10, ebiten.MouseButtonLeft, control.ActionPress)
	r.handle(press)
	if got := r.drained(); len(got) != 0 {
		t.Fatalf("a press alone issued %v", got)
	}
	start, current, dragging := r.local.DragBox()
	if !dragging || start != geom.NewVec(10, 10) || current != geom.NewVec(10, 10) {
		t.Errorf("after the press: start %v current %v dragging %v, want (10,10) (10,10) true", start, current, dragging)
	}

	r.handle(&control.InputEvents{MousePos: geom.NewVec(40, 60)})
	if start, current, dragging := r.local.DragBox(); !dragging || start != geom.NewVec(10, 10) || current != geom.NewVec(40, 60) {
		t.Errorf("mid-drag: start %v current %v dragging %v, want (10,10) (40,60) true", start, current, dragging)
	}

	release := &control.InputEvents{MousePos: geom.NewVec(60, 60)}
	release.AddClickEvent(60, 60, ebiten.MouseButtonLeft, control.ActionRelease)
	r.handle(release)
	if got := r.drained(); len(got) != 1 || got[0].Command.Cell != 10*1000+60 {
		t.Errorf("the release issued %v, want one order from 10 to 60", got)
	}
	if _, _, dragging := r.local.DragBox(); dragging {
		t.Error("still dragging after the release")
	}
}

func TestWorldBox_StaysNarrowAcrossATorusSeam(t *testing.T) {
	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 1000, Height: 1000, Edges: aabbworld.Torus},
		Entities: world.EntitiesCfg{MaxCount: 1, MinSize: 1, MaxSize: 10},
	})
	w.Res.Camera = camera.NewFromSpaceWithConfig(1000, 1000, aabbworld.Torus, camera.Config{ViewportWidth: 200, ViewportHeight: 200})
	w.Res.Camera.MoveTo(950, 500)
	p := players.NewPlugin(w)
	ctx := players.Context{Player: p.Local("tester")}

	box := ctx.WorldBox(geom.NewVec(0, 0), geom.NewVec(200, 10))
	if width := box.BottomRight.X - box.TopLeft.X; width > 200 {
		t.Errorf("WorldBox is %v wide across the seam, want the 200 that was dragged", width)
	}
}

// cameraRig is a rig with the default camera bindings, started, so Pan and Zoom reach the camera.
func cameraRig(t *testing.T, cfg ...camera.Config) (*rig, *goke.ECS) {
	t.Helper()
	r := newRig(t, cfg...)
	r.bind(players.CameraBindings()...)
	return r, r.start()
}

func (r *rig) move(ev *control.InputEvents, ecs *goke.ECS) {
	r.handle(ev)
	ecs.Tick(time.Second / 60)
}

func TestCamera_WheelZooms(t *testing.T) {
	r, ecs := cameraRig(t)
	cam := r.local.Camera
	r.move(&control.InputEvents{ScrollDelta: 1}, ecs)
	if cam.Zoom() <= 1 {
		t.Fatalf("Zoom() after a notch up = %v, want > 1", cam.Zoom())
	}
	zoomedIn := cam.Zoom()
	r.move(&control.InputEvents{ScrollDelta: -1}, ecs)
	if cam.Zoom() >= zoomedIn {
		t.Fatalf("Zoom() after a notch down = %v, want < %v", cam.Zoom(), zoomedIn)
	}
}

func TestCamera_MiddleDragPansOneToOneWithTheCursor(t *testing.T) {
	for _, zoom := range []float32{1, 2} {
		r, ecs := cameraRig(t, camera.Config{ViewportWidth: 200, ViewportHeight: 200})
		cam := r.local.Camera
		cam.MoveTo(400, 400)
		cam.ZoomIn(zoom, 500, 500)
		before := cam.Bounds()

		r.move(&control.InputEvents{MousePos: geom.NewVec(100, 100), MiddleDown: true, CursorDelta: geom.NewVec(10, 0)}, ecs)

		after := cam.Bounds()
		if want := before.TopLeft.X - 10/float64(zoom); after.TopLeft.X != want {
			t.Errorf("zoom %v: TopLeft.X after a 10-pixel drag = %v, want %v", zoom, after.TopLeft.X, want)
		}
	}
}

func TestCamera_EdgeScrollIsTheSameOnScreenAtAnyZoom(t *testing.T) {
	for _, zoom := range []float32{1, 2, 4} {
		r, ecs := cameraRig(t, camera.Config{ViewportWidth: 200, ViewportHeight: 200})
		cam := r.local.Camera
		cam.MoveTo(400, 400)
		cam.ZoomIn(zoom, 500, 500)
		before := cam.Bounds()

		r.move(&control.InputEvents{MousePos: geom.NewVec(190, 100)}, ecs) // near the right edge of the 200-pixel window

		moved := (cam.Bounds().TopLeft.X - before.TopLeft.X) * float64(zoom)
		if moved != float64(players.DefaultScrollSpeed) {
			t.Errorf("zoom %v: an edge scroll moved %v pixels of world, want %v", zoom, moved, players.DefaultScrollSpeed)
		}
	}
}

func TestCamera_EdgeDeadZoneAndWindowEdges(t *testing.T) {
	cases := map[string]struct {
		ev    control.InputEvents
		moves bool
	}{
		"near the right edge":               {control.InputEvents{MousePos: geom.NewVec(990, 500)}, true},
		"at the true edge":                  {control.InputEvents{MousePos: geom.NewVec(999, 500)}, false},
		"at the true edge, window fills it": {control.InputEvents{MousePos: geom.NewVec(999, 500), WindowFillsScreen: true}, true},
		"outside the window":                {control.InputEvents{MousePos: geom.NewVec(-5, 500)}, false},
		"middle drag outside the window":    {control.InputEvents{MousePos: geom.NewVec(-5, 500), MiddleDown: true, CursorDelta: geom.NewVec(10, 0)}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, ecs := cameraRig(t)
			cam := r.local.Camera
			cam.ZoomIn(2, 500, 500)
			before := cam.Bounds()
			ev := tc.ev
			r.move(&ev, ecs)
			if moved := cam.Bounds() != before; moved != tc.moves {
				t.Errorf("camera moved = %v, want %v", moved, tc.moves)
			}
		})
	}
}

func TestPlugin_Contract(t *testing.T) {
	r := newRig(t)
	if r.p.Name() != "gram.players" {
		t.Errorf("Name = %q", r.p.Name())
	}
	if r.p.EventHandler() == nil {
		t.Error("EventHandler is nil — players translate input")
	}
	if r.p.Renderer() != nil || r.p.Serializable() != nil {
		t.Error("players draw nothing and save nothing of their own")
	}
	if err := r.p.RegisterBehavior(struct{}{}); !errors.Is(err, plugin.ErrUnhostedBehavior) {
		t.Errorf("RegisterBehavior = %v, want ErrUnhostedBehavior", err)
	}
}
