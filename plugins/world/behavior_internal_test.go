package world

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
)

// behaviorTag marks the one kind TestBehavior_Include applies to.
type behaviorTag struct{}

// driveVelocity is a Behavior that sets every entity it visits moving, counts
// them, and notes the order it ran in.
type driveVelocity struct {
	tagged bool
	log    *[]string
	name   string

	visited int
	query   *goke.Query
	vel     goke.Comp[Velocity]
}

func (b *driveVelocity) Init(si *goke.SysInit) {
	qb := si.NewQueryBuilder(&b.vel)
	if b.tagged {
		qb.Include(goke.Include[behaviorTag]())
	}
	b.query = qb.Build()
}

func (b *driveVelocity) Update(*goke.CmdBuf, time.Duration) {
	if b.log != nil {
		*b.log = append(*b.log, b.name)
	}
	b.query.All()
	for b.query.Next() {
		cursor := b.query.Cursor()
		vels := b.vel.Slice(cursor)
		for i := range cursor.IDs {
			vels[i].Dir = geom.NewVec(1.0, 0.0)
			vels[i].Value = 600
			b.visited++
		}
	}
}

func spawnAt(wm *module, x float64, extras ...ComponentTemplate) {
	pos := Position{AABB: plane.NewAABB(geom.NewVec(x, 100), 10, 10)}
	wm.populate(EntKind{
		Name:       "e",
		Position:   Const(pos),
		Velocity:   Const(Velocity{}),
		Components: extras,
	}, []any{nil})
}

// tickWorld runs one full world tick — behaviors, then velocity, then movement
// — through the module's own RunPlan, with no engine or window involved.
func tickWorld(t *testing.T, wm *module) []float64 {
	t.Helper()

	var pos goke.Comp[Position]
	var query *goke.Query
	ecs := goke.New()
	ecs.Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: func(si *goke.SysInit) {
		query = si.NewQueryBuilder(&pos).Build()
	}})...)
	wm.RegSystems(ecs)
	ecs.SetPlan(wm.RunPlan)
	ecs.Tick(time.Second / 10)

	var xs []float64
	query.All()
	for query.Next() {
		cursor := query.Cursor()
		positions := pos.Slice(cursor)
		for i := range cursor.IDs {
			xs = append(xs, positions[i].TopLeft.X)
		}
	}
	return xs
}

// A decision taken this tick has to reach this tick's movement — that is the
// whole reason the phase sits before velocity and move.
func TestBehavior_RunsBeforeMovement(t *testing.T) {
	p := testPlugin()
	p.RegisterBehavior(&driveVelocity{})
	wm := p.module
	spawnAt(wm, 100)

	xs := tickWorld(t, wm)
	if len(xs) != 1 {
		t.Fatalf("found %d entities, want 1", len(xs))
	}
	if xs[0] <= 100 {
		t.Errorf("entity sits at x=%v after a tick, want it moved — the behavior ran too late to be integrated", xs[0])
	}
}

func TestBehavior_RunsInRegistrationOrder(t *testing.T) {
	var log []string
	wm := testWorld()
	wm.RegisterBehavior(&driveVelocity{log: &log, name: "first"})
	wm.RegisterBehavior(&driveVelocity{log: &log, name: "second"})
	spawnAt(wm, 100)

	tickWorld(t, wm)
	if len(log) != 2 || log[0] != "first" || log[1] != "second" {
		t.Errorf("behaviors ran as %v, want [first second]", log)
	}
}

// Include restricts a behavior to the archetypes carrying its tag, which is
// what keeps a rare behavior from walking the whole world.
func TestBehavior_IncludeVisitsOnlyTaggedEntities(t *testing.T) {
	b := &driveVelocity{tagged: true}
	wm := testWorld()
	wm.RegisterBehavior(b)
	spawnAt(wm, 100)
	spawnAt(wm, 300, Const(behaviorTag{}))

	tickWorld(t, wm)
	if b.visited != 1 {
		t.Errorf("behavior visited %d entities, want only the tagged one", b.visited)
	}
}

// A world nobody registered a behavior with must tick exactly as before.
func TestBehavior_NoneRegisteredLeavesTheTickUnchanged(t *testing.T) {
	wm := testWorld()
	spawnAt(wm, 100)

	if xs := tickWorld(t, wm); len(xs) != 1 || xs[0] != 100 {
		t.Errorf("positions = %v, want the entity still at 100", xs)
	}
}
