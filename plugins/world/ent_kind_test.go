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

type propData struct{ x uint32 }

func spawnerTestPos() Position {
	return Position{AABB: plane.NewAABB(geom.NewVec[uint32](0, 0), 10, 10)}
}

// statKind defines a kind at a fixed Position whose spawnerStat is read from int roster data.
func statKind(k Kind[int]) EntKind {
	return EntKind{
		Position:   k.Const(spawnerTestPos()),
		Velocity:   k.Const(Velocity{}),
		Components: []ComponentTemplate{k.Load(func(hp int) spawnerStat { return spawnerStat{HP: hp} })},
	}
}

// setupWorld runs wm's queued spawns in a fresh ECS, then onInit.
func setupWorld(wm *module, onInit func(si *goke.SysInit)) {
	goke.New().Setup(append(wm.SetupSystems(), goke.SystemFn{OnInit: onInit})...)
}

func testPlugin() *Plugin { return NewPlugin(testWorld().config) }

func TestEntKindDict_Define_AssignsSpriteIDsByOrder(t *testing.T) {
	dict := newEntKindDict()
	dict.Define("red", statKind)
	dict.Define("blue", statKind)

	red, ok := dict.Get("red")
	if !ok || red.SpriteID != 0 {
		t.Errorf("Get(%q) SpriteID = %v (found %v), want 0", "red", red.SpriteID, ok)
	}
	blue, ok := dict.Get("blue")
	if !ok || blue.SpriteID != 1 {
		t.Errorf("Get(%q) SpriteID = %v (found %v), want 1", "blue", blue.SpriteID, ok)
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

func TestEntKindDict_Entry_PanicsOnUnknownKind(t *testing.T) {
	dict := newEntKindDict()
	dict.Define("unit", statKind)
	defer func() {
		if recover() == nil {
			t.Error("expected Entry to panic")
		}
	}()
	dict.Entry("ghost", 1)
}

func TestEntKindDict_Entry_PanicsOnBadEntry(t *testing.T) {
	bare := func(k EntKind) func(Kind[int]) EntKind { return func(Kind[int]) EntKind { return k } }
	cases := map[string]struct {
		kind func(Kind[int]) EntKind
		data any
	}{
		"wrong data":  {kind: statKind, data: "x"},
		"no Position": {kind: bare(EntKind{Velocity: Const(Velocity{})}), data: 1},
		"no Velocity": {kind: bare(EntKind{Position: Const(spawnerTestPos())}), data: 1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dict := newEntKindDict()
			dict.Define("unit", tc.kind)
			defer func() {
				if recover() == nil {
					t.Error("expected Entry to panic")
				}
			}()
			dict.Entry("unit", tc.data)
		})
	}
}

func TestPopulate_ConstAndLoadComponents(t *testing.T) {
	wm := testWorld()
	kind := statKind(Kind[int]{})
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
		t.Errorf("HP per entity = %v, want [9 4] (read from each entry's data)", hps)
	}
}

func TestPopulate_WithEffect_RunsAfterWriteWithValueAndID(t *testing.T) {
	wm := testWorld()
	var gotHP, calls int
	var gotID uid.UID64
	var k Kind[int]
	kind := statKind(k)
	kind.Components = []ComponentTemplate{
		k.Load(func(hp int) spawnerStat { return spawnerStat{HP: hp} }).
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

func TestPopulate_KindsWithDifferentDataAndComponents(t *testing.T) {
	p := testPlugin()
	kinds := p.EntKindDict()
	kinds.Define("unit", statKind)
	kinds.Define("prop", func(k Kind[propData]) EntKind {
		return EntKind{
			Position: k.Load(func(d propData) Position {
				return Position{AABB: plane.NewAABB(geom.NewVec(d.x, 0), 10, 10)}
			}),
			Velocity:   k.Const(Velocity{}),
			Components: []ComponentTemplate{k.Const(spawnerTag{})},
		}
	})
	p.Seed(kinds.Entry("unit", 5), kinds.Entry("prop", propData{x: 40}), kinds.Entry("unit", 6))
	if err := p.Populate(); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	var stat goke.Comp[spawnerStat]
	var tagPos goke.Comp[Position]
	var units int
	var propX []uint32
	setupWorld(p.module, func(si *goke.SysInit) {
		uq := si.NewQueryBuilder(&stat).Build()
		for uq.All(); uq.Next(); {
			units += len(uq.Cursor().IDs)
		}
		pq := si.NewQueryBuilder(&tagPos).Include(goke.Include[spawnerTag]()).Build()
		for pq.All(); pq.Next(); {
			for _, pos := range tagPos.Slice(pq.Cursor()) {
				propX = append(propX, pos.TopLeft.X)
			}
		}
	})

	if units != 2 {
		t.Errorf("units = %d, want 2", units)
	}
	if len(propX) != 1 || propX[0] != 40 {
		t.Errorf("prop positions X = %v, want [40] (read from its own data type)", propX)
	}
}

func TestPlugin_Populate_ZeroEntryErrorsWithoutSpawning(t *testing.T) {
	p := testPlugin()
	p.EntKindDict().Define("unit", statKind)
	p.Seed(p.EntKindDict().Entry("unit", 1), Entry{})

	if err := p.Populate(); err == nil {
		t.Fatal("Populate: expected an error for an Entry not built by EntKindDict.Entry")
	}
	if n := len(p.module.SetupSystems()); n != 0 {
		t.Errorf("queued %d spawns, want 0", n)
	}
}

// Kind.Const is the in-builder spelling of Const; both must yield the same
// template, since one delegates to the other.
func TestKind_Const_MatchesPackageConst(t *testing.T) {
	var k Kind[int]
	viaKind, viaPkg := k.Const(spawnerStat{HP: 3}), Const(spawnerStat{HP: 3})

	if got := viaKind.resolve(nil, 0); got != viaPkg.resolve(nil, 0) {
		t.Errorf("k.Const resolved to %+v, Const to %+v", got, viaPkg.resolve(nil, 0))
	}
	// The roster entry is ignored either way — that is what makes it constant.
	if got := viaKind.resolve(99, 0); got.HP != 3 {
		t.Errorf("k.Const read the roster entry: HP = %d, want 3", got.HP)
	}
}
