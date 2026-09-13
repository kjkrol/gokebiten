package world

import (
	"testing"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokg/geom"
	"github.com/kjkrol/gokg/plane"
	"github.com/kjkrol/uid"
)

type spawnerTag struct{}

type spawnerStat struct{ HP int }

func spawnerTestPos() Position {
	return Position{AABB: plane.NewAABB(geom.NewVec[uint32](0, 0), 10, 10)}
}

// statKind is an EntKind whose Position is fixed and whose spawnerStat is read from int Data.
func statKind(name string) EntKind {
	return EntKind{
		Name:       name,
		Position:   Const(spawnerTestPos()),
		Velocity:   Const(Velocity{}),
		Components: []ComponentTemplate{Load(func(hp int) spawnerStat { return spawnerStat{HP: hp} })},
	}
}

// setupWorld runs wm's queued spawns in a fresh ECS, then onInit.
func setupWorld(wm *module, onInit func(si *goke.SysInit)) {
	goke.New().Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: onInit})...)
}

func TestEntKindDict_Create_AssignsSpriteIDsByOrder(t *testing.T) {
	dict := newEntKindDict()
	dict.Create(EntKind{Name: "red"}, EntKind{Name: "blue"})

	red, ok := dict.Get("red")
	if !ok || red.SpriteID != 0 {
		t.Errorf("Get(%q) = %+v, %v, want SpriteID 0", "red", red, ok)
	}
	blue, ok := dict.Get("blue")
	if !ok || blue.SpriteID != 1 {
		t.Errorf("Get(%q) = %+v, %v, want SpriteID 1", "blue", blue, ok)
	}
	if got := len(dict.All()); got != 2 {
		t.Errorf("len(All()) = %d, want 2", got)
	}
}

func TestEntKindDict_Get_UnknownName(t *testing.T) {
	if _, ok := newEntKindDict().Get("nope"); ok {
		t.Error("Get(unknown) ok = true, want false")
	}
}

func TestPopulate_ConstAndLoadComponents(t *testing.T) {
	wm := testWorld()
	kind := statKind("red")
	kind.SpriteID = 7
	kind.Components = append(kind.Components, Const(spawnerTag{}))
	wm.populate(kind, []any{9, 4})

	var appearance goke.Comp[Appearance]
	var stat goke.Comp[spawnerStat]
	var q *goke.Query
	setupWorld(wm, func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&appearance, &stat).Include(goke.Include[spawnerTag]()).Build()
	})

	var hps []int
	q.All()
	for q.Next() {
		cur := q.Cursor()
		appearances, stats := appearance.Slice(cur), stat.Slice(cur)
		for i := range cur.IDs {
			if appearances[i].SpriteID != 7 {
				t.Errorf("SpriteID = %v, want 7", appearances[i].SpriteID)
			}
			hps = append(hps, stats[i].HP)
		}
	}
	if len(hps) != 2 || hps[0] != 9 || hps[1] != 4 {
		t.Errorf("HP per entity = %v, want [9 4] (read from each entry's Data)", hps)
	}
}

func TestPopulate_WithEffect_RunsAfterWriteWithValueAndID(t *testing.T) {
	wm := testWorld()
	var gotHP, calls int
	var gotID uid.UID64
	kind := statKind("red")
	kind.Components = []ComponentTemplate{
		Load(func(hp int) spawnerStat { return spawnerStat{HP: hp} }).
			WithEffect(func(v spawnerStat, id uid.UID64) {
				calls++
				gotHP, gotID = v.HP, id
			}),
	}
	wm.populate(kind, []any{9})

	var stat goke.Comp[spawnerStat]
	var wantID uid.UID64
	setupWorld(wm, func(si *goke.SysInit) {
		q := si.NewQueryBuilder(&stat).Build()
		q.All()
		for q.Next() {
			for _, id := range q.Cursor().IDs {
				wantID = id
			}
		}
	})

	if calls != 1 || gotHP != 9 || gotID != wantID {
		t.Errorf("effect calls=%d HP=%d id=%v, want 1, 9, %v", calls, gotHP, gotID, wantID)
	}
}

func TestPopulate_KindsWithDifferentComponentSets(t *testing.T) {
	p := NewPlugin(testWorld().config)
	p.EntKindDict().Create(
		statKind("unit"),
		EntKind{Name: "prop", Position: Const(spawnerTestPos()), Velocity: Const(Velocity{}),
			Components: []ComponentTemplate{Const(spawnerTag{})}},
	)
	p.Seed(Roster{{Kind: "unit", Data: 5}, {Kind: "prop"}, {Kind: "unit", Data: 6}})
	if err := p.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	var stat goke.Comp[spawnerStat]
	var units, props int
	setupWorld(p.module, func(si *goke.SysInit) {
		uq := si.NewQueryBuilder(&stat).Build()
		for uq.All(); uq.Next(); {
			units += len(uq.Cursor().IDs)
		}
		pq := si.NewQueryBuilder().Include(goke.Include[spawnerTag]()).Build()
		for pq.All(); pq.Next(); {
			props += len(pq.Cursor().IDs)
		}
	})

	if units != 2 || props != 1 {
		t.Errorf("units=%d props=%d, want 2 and 1", units, props)
	}
}

func TestPlugin_Populate_RejectsBadRosterWithoutSpawning(t *testing.T) {
	cases := map[string]struct {
		kinds  []EntKind
		roster Roster
	}{
		"unknown kind": {kinds: []EntKind{statKind("unit")}, roster: Roster{{Kind: "unit", Data: 1}, {Kind: "ghost", Data: 1}}},
		"wrong Data":   {kinds: []EntKind{statKind("unit")}, roster: Roster{{Kind: "unit", Data: 1}, {Kind: "unit", Data: "x"}}},
		"no Position":  {kinds: []EntKind{{Name: "bare", Velocity: Const(Velocity{})}}, roster: Roster{{Kind: "bare"}}},
		"no Velocity":  {kinds: []EntKind{{Name: "bare", Position: Const(spawnerTestPos())}}, roster: Roster{{Kind: "bare"}}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := NewPlugin(testWorld().config)
			p.EntKindDict().Create(tc.kinds...)
			p.Seed(tc.roster)

			if err := p.Populate(); err == nil {
				t.Fatal("Populate: expected an error, got nil")
			}
			if n := len(p.module.SetupSystems()); n != 0 {
				t.Errorf("queued %d spawns, want 0", n)
			}
		})
	}
}
