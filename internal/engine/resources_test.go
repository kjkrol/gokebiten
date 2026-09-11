package engine

import "testing"

type persistedResource struct{ N int }

func (p *persistedResource) Persisted() []any { return []any{&p.N} }

func TestStorage_Register_KeyedByName(t *testing.T) {
	s := newStorage()
	s.register("a", &persistedResource{N: 5})

	got, ok := s.persisted()["a"]
	if !ok || len(got) != 1 || *(got[0].(*int)) != 5 {
		t.Errorf("persisted()[%q] = %v, ok=%v, want [5], true", "a", got, ok)
	}
}

func TestStorage_Persisted_AggregatesByName(t *testing.T) {
	s := newStorage()
	s.register("a", &persistedResource{N: 1})
	s.register("b", &persistedResource{N: 2})

	got := s.persisted()
	if len(got) != 2 {
		t.Fatalf("persisted() returned %d entries, want 2", len(got))
	}
	if *(got["a"][0].(*int)) != 1 {
		t.Errorf(`persisted()["a"] = %v, want [1]`, got["a"])
	}
	if *(got["b"][0].(*int)) != 2 {
		t.Errorf(`persisted()["b"] = %v, want [2]`, got["b"])
	}
}

func TestStorage_Register_OverwritesSameName(t *testing.T) {
	s := newStorage()
	s.register("a", &persistedResource{N: 1})
	s.register("a", &persistedResource{N: 2})

	got := s.persisted()["a"]
	if *(got[0].(*int)) != 2 {
		t.Errorf(`persisted()["a"] = %v, want [2] (last register should win)`, got)
	}
}

type unencodableResource struct{ fn func() }

func (r *unencodableResource) Persisted() []any { return []any{&r.fn} }

func TestStorage_Register_PanicsOnUnencodablePersisted(t *testing.T) {
	s := newStorage()
	defer func() {
		if recover() == nil {
			t.Fatal("expected register to panic when Persisted() returns an unencodable value")
		}
	}()
	s.register("a", &unencodableResource{})
}
