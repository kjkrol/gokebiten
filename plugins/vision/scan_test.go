package vision_test

import (
	"math"
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/vision"
	"github.com/kjkrol/gokebiten/plugins/world"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

// installCtx is the plugin.Installer a Stage would hand over, minus the engine.
type installCtx struct {
	ecs     *goke.ECS
	pending []func() []goke.System
	tracked []any
}

func (c *installCtx) UseModule(m goke.Module) {
	regSys := goke.SystemFn{OnInit: func(*goke.SysInit) { m.RegSystems(c.ecs) }}
	c.tracked = append(c.tracked, m)
	c.pending = append(c.pending, func() []goke.System { return append(m.SetupSystems(), regSys) })
}
func (c *installCtx) Setup(providers ...goke.SetupProvider) {
	for _, p := range providers {
		c.tracked = append(c.tracked, p)
		c.pending = append(c.pending, p.SetupSystems)
	}
}
func (c *installCtx) RegSys(factory func() goke.System) goke.Runnable { return c.ecs.RegSys(factory()) }
func (c *installCtx) ECS() *goke.ECS                                  { return c.ecs }

// spawn describes one entity the fixture puts in the world.
type spawn struct {
	x, y    float64
	sight   *vision.Sight // nil for something that is merely seen
	outline bool
}

func at(x, y float64) world.Position {
	return world.Position{AABB: plane.NewAABB(geom.NewVec(x, y), 10, 10)}
}

// scene installs world+vision, spawns everything, and ticks once. It returns
// the observers' ids paired with what they saw.
func scene(t *testing.T, spawns ...spawn) ([]uid.UID64, []vision.Sighted, []vision.SightOutline) {
	t.Helper()

	w := world.NewPlugin(world.Config{
		Space:    world.SpaceCfg{Width: 2000, Height: 2000},
		Entities: world.EntitiesCfg{MaxCount: 64, MinSize: 1, MaxSize: 100},
	})
	v := vision.NewPlugin(w)

	ctx := &installCtx{ecs: goke.New()}
	if err := w.Install(ctx); err != nil {
		t.Fatalf("world Install: %v", err)
	}
	if err := v.Install(ctx); err != nil {
		t.Fatalf("vision Install: %v", err)
	}

	dict := w.EntKindDict()
	for i, s := range spawns {
		comps := []world.ComponentTemplate{}
		if s.sight != nil {
			comps = append(comps, world.Const(*s.sight), world.Const(vision.Sighted{}))
			if s.outline {
				comps = append(comps, world.Const(vision.SightOutline{}))
			}
		}
		dict.Define(kindName(i), func(k world.Kind[spawn]) world.EntKind {
			return world.EntKind{
				Position:   k.Load(func(d spawn) world.Position { return at(d.x, d.y) }),
				Velocity:   k.Const(world.Velocity{}),
				Components: comps,
			}
		})
		w.Seed(dict.Entry(kindName(i), s))
	}
	if err := w.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	var sightedComp goke.Comp[vision.Sighted]
	var outlineComp goke.OptComp[vision.SightOutline]
	var query *goke.Query
	var systems []goke.System
	for _, produce := range ctx.pending {
		systems = append(systems, produce()...)
	}
	systems = append(systems, goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&sightedComp).Optional(&outlineComp).Build()
	}})
	ctx.ecs.Setup(systems...)

	ctx.ecs.SetPlan(func(rc goke.RunCtx, d time.Duration) {
		w.RunPlan(rc, d)
		v.RunPlan(rc, d)
	})
	ctx.ecs.Tick(time.Second / 60)

	var ids []uid.UID64
	var seen []vision.Sighted
	var outlines []vision.SightOutline
	query.All()
	for query.Next() {
		cursor := query.Cursor()
		got := sightedComp.Slice(cursor)
		var outs []vision.SightOutline
		if outlineComp.Present(cursor) {
			outs = outlineComp.Slice(cursor)
		}
		for i, id := range cursor.IDs {
			ids = append(ids, id)
			seen = append(seen, got[i])
			if outs != nil {
				outlines = append(outlines, outs[i])
			}
		}
	}
	return ids, seen, outlines
}

func kindName(i int) string { return string(rune('a' + i)) }

func eastward(half, radius float64) *vision.Sight {
	return &vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: half, Radius: radius}
}

func TestScan_ReportsWhatIsInTheConeNearestFirst(t *testing.T) {
	_, seen, _ := scene(t,
		spawn{x: 500, y: 500, sight: eastward(math.Pi/4, 600)},
		spawn{x: 900, y: 560}, // further, and off the near one's shadow
		spawn{x: 700, y: 500}, // nearer
	)

	if len(seen) != 1 {
		t.Fatalf("%d entities carry Sighted, want 1", len(seen))
	}
	if seen[0].Count != 2 {
		t.Fatalf("saw %d entities, want 2", seen[0].Count)
	}
	if seen[0].Dists[0] >= seen[0].Dists[1] {
		t.Errorf("distances %v, %v are not nearest-first", seen[0].Dists[0], seen[0].Dists[1])
	}
}

func TestScan_IgnoresWhatFallsOutsideTheCone(t *testing.T) {
	_, seen, _ := scene(t,
		spawn{x: 500, y: 500, sight: eastward(math.Pi/8, 600)},
		spawn{x: 700, y: 500}, // straight ahead
		spawn{x: 500, y: 900}, // off to the side
		spawn{x: 200, y: 500}, // behind
	)

	if seen[0].Count != 1 {
		t.Errorf("saw %d entities, want only the one ahead", seen[0].Count)
	}
}

func TestScan_KeepsAtMostMaxSeen(t *testing.T) {
	spawns := []spawn{{x: 200, y: 500, sight: eastward(math.Pi/3, 900)}}
	for i := range vision.MaxSeen + 4 {
		spawns = append(spawns, spawn{x: float64(400 + i*40), y: float64(480 + i*8)})
	}

	_, seen, _ := scene(t, spawns...)
	if int(seen[0].Count) > vision.MaxSeen {
		t.Errorf("recorded %d sightings, want at most MaxSeen (%d)", seen[0].Count, vision.MaxSeen)
	}
	if seen[0].Count == 0 {
		t.Fatal("recorded nothing, so the cap was never under pressure")
	}
}

// An entity without SightOutline is still scanned — the component marks what to
// draw, not what to perceive.
func TestScan_WorksWithoutAnOutline(t *testing.T) {
	_, seen, outlines := scene(t,
		spawn{x: 500, y: 500, sight: eastward(math.Pi/4, 600)},
		spawn{x: 700, y: 500},
	)

	if len(outlines) != 0 {
		t.Errorf("%d outlines computed, want none", len(outlines))
	}
	if seen[0].Count != 1 {
		t.Errorf("saw %d entities without an outline, want 1", seen[0].Count)
	}
}

func TestScan_FillsTheOutlineWhenAsked(t *testing.T) {
	_, _, outlines := scene(t,
		spawn{x: 500, y: 500, sight: eastward(math.Pi/6, 300), outline: true},
		spawn{x: 700, y: 500},
	)

	if len(outlines) != 1 {
		t.Fatalf("%d outlines, want 1", len(outlines))
	}
	o := outlines[0]
	if o.Count < 2 {
		t.Fatalf("outline has %d samples, want at least the two cone edges", o.Count)
	}
	blocked := false
	for i := range int(o.Count) {
		if o.Depths[i] <= 0 || o.Depths[i] > 300 {
			t.Fatalf("sample %d reaches %v, outside (0,300]", i, o.Depths[i])
		}
		blocked = blocked || o.Depths[i] < 300
	}
	if !blocked {
		t.Error("nothing shortened the outline, though something stands straight ahead")
	}
}

// A cone wider and longer than the buffer was sized for must still fit it.
func TestScan_OutlineNeverOverrunsItsBuffer(t *testing.T) {
	huge := &vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: math.Pi/2 - 0.01, Radius: 1900}
	_, _, outlines := scene(t,
		spawn{x: 50, y: 500, sight: huge, outline: true},
		spawn{x: 700, y: 500},
	)

	if int(outlines[0].Count) > vision.MaxSamples {
		t.Errorf("outline holds %d samples, want at most MaxSamples (%d)", outlines[0].Count, vision.MaxSamples)
	}
}

// MaxSamples is derived, not chosen. Pinning it from both sides says what it
// means without restating the arithmetic: enough to hold EdgeTolerance across
// the widest cone the constants describe, and not one sample more.
func TestMaxSamples_IsTheSmallestThatHoldsTheTolerance(t *testing.T) {
	half := float64(vision.MaxHalfAngleMilli) / 1000
	arc := vision.MaxSightRadius * 2 * half

	if drift := arc / float64(vision.MaxSamples-1); drift > vision.EdgeTolerance {
		t.Errorf("a full-size cone drifts %.3f units at full range, over EdgeTolerance (%d)", drift, vision.EdgeTolerance)
	}
	if drift := arc / float64(vision.MaxSamples-2); drift <= vision.EdgeTolerance {
		t.Errorf("one sample fewer still drifts only %.3f — MaxSamples (%d) is bigger than it needs to be", drift, vision.MaxSamples)
	}
}

// A cone the raycaster refuses — zero width, zero reach — leaves the entity
// seeing nothing rather than keeping stale sightings.
func TestScan_ClearsSightedWhenTheConeIsUnanswerable(t *testing.T) {
	blind := &vision.Sight{Facing: geom.NewVec(1.0, 0.0), HalfAngle: 0, Radius: 0}
	_, seen, _ := scene(t,
		spawn{x: 500, y: 500, sight: blind},
		spawn{x: 700, y: 500},
	)

	if seen[0].Count != 0 {
		t.Errorf("a blind entity recorded %d sightings, want none", seen[0].Count)
	}
}
