package kind_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kjkrol/gram/plugins/world/kind"
	"github.com/kjkrol/gram/render"
	"github.com/kjkrol/uid"
)

type row struct{ hp int }

type stat struct{ HP int }

// shelf is the least a Registry can be: it remembers what it was given.
type shelf struct {
	name string
	row  reflect.Type
	spec kind.Spec
}

func (s *shelf) Register(name string, row reflect.Type, spec kind.Spec) (kind.ID, render.SpriteID) {
	s.name, s.row, s.spec = name, row, spec
	return 3, 7
}

func TestConst_IsTheSameForEveryRow(t *testing.T) {
	c := kind.Const(stat{HP: 3})

	for _, r := range []any{nil, row{hp: 99}} {
		if got := c.Resolve(r, 0); got.HP != 3 {
			t.Errorf("Const resolved to %+v for row %v, want HP 3 whatever the row", got, r)
		}
	}
}

func TestLoad_ReadsTheRow_AndRunsItsEffectWithTheValue(t *testing.T) {
	var seen stat
	var seenID uid.UID64
	c := kind.Load(func(r row) stat { return stat{HP: r.hp} }).
		WithEffect(func(v stat, id uid.UID64) { seen, seenID = v, id })

	got := c.Resolve(row{hp: 9}, uid.UID64(42))

	if got.HP != 9 {
		t.Errorf("Load resolved to %+v, want the row's 9", got)
	}
	if seen.HP != 9 || seenID != 42 {
		t.Errorf("effect saw %+v for entity %v, want the value 9 and the entity 42", seen, seenID)
	}
}

func TestDefine_RegistersTheSpecAndHandsBackTheKind(t *testing.T) {
	reg := &shelf{}
	spec := kind.Spec{kind.Load(func(r row) stat { return stat{HP: r.hp} }), kind.Const(struct{}{})}

	unit := kind.Define[row](reg, "unit", spec)

	if reg.name != "unit" || reg.row != reflect.TypeFor[row]() || len(reg.spec) != 2 {
		t.Errorf("the registry was given (%q, %v, %d comps), want (unit, row, 2)", reg.name, reg.row, len(reg.spec))
	}
	if unit.Name() != "unit" || unit.ID() != 3 || unit.SpriteID() != 7 {
		t.Errorf("the kind says (%q, %d, %d), want what the registry assigned: (unit, 3, 7)", unit.Name(), unit.ID(), unit.SpriteID())
	}

	entry := unit.Entry(row{hp: 5})
	if entry.Kind() != "unit" || entry.Row() != (row{hp: 5}) {
		t.Errorf("entry = (%q, %v), want (unit, {5})", entry.Kind(), entry.Row())
	}
}

func TestDefine_RefusesALoadThatReadsAnotherRowType(t *testing.T) {
	defer func() {
		msg, _ := recover().(string)
		for _, want := range []string{`"unit"`, "kind_test.row", "int"} {
			if !strings.Contains(msg, want) {
				t.Errorf("panic %q does not mention %s", msg, want)
			}
		}
	}()

	kind.Define[row](&shelf{}, "unit", kind.Spec{kind.Load(func(hp int) stat { return stat{HP: hp} })})
	t.Error("Define accepted a Load over int for a kind whose rows are row")
}
