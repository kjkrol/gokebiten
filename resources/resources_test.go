package resources

import "testing"

type testResourceA struct{ N int }
type testResourceB struct{ S string }

func (*testResourceA) Resources() {}
func (*testResourceB) Resources() {}

func TestStorage_InsertGet_RoundTrip(t *testing.T) {
	s := NewStorage()
	s.Insert(&testResourceA{N: 7})

	got := s.Get[*testResourceA]()
	if got.N != 7 {
		t.Errorf("Get[*testResourceA]().N = %d, want 7", got.N)
	}
}

func TestStorage_TryGet_MissingReturnsFalse(t *testing.T) {
	s := NewStorage()
	_, ok := s.TryGet[*testResourceA]()
	if ok {
		t.Error("TryGet on an empty registry: ok = true, want false")
	}
}

func TestStorage_Get_MissingPanics(t *testing.T) {
	s := NewStorage()
	defer func() {
		if recover() == nil {
			t.Fatal("expected Get to panic when the resource isn't registered")
		}
	}()
	s.Get[*testResourceA]()
}

func TestStorage_Insert_OverwritesPreviousValue(t *testing.T) {
	s := NewStorage()
	s.Insert(&testResourceA{N: 1})
	s.Insert(&testResourceA{N: 2})

	got := s.Get[*testResourceA]()
	if got.N != 2 {
		t.Errorf("Get[*testResourceA]().N = %d, want 2 (last Insert should win)", got.N)
	}
}

func TestStorage_DifferentTypesDoNotCollide(t *testing.T) {
	s := NewStorage()
	s.Insert(&testResourceA{N: 1})
	s.Insert(&testResourceB{S: "hi"})

	if got := s.Get[*testResourceA](); got.N != 1 {
		t.Errorf("testResourceA.N = %d, want 1", got.N)
	}
	if got := s.Get[*testResourceB](); got.S != "hi" {
		t.Errorf("testResourceB.S = %q, want %q", got.S, "hi")
	}
}

type resettableResource struct{ resetCalls int }

func (r *resettableResource) Reset()   { r.resetCalls++ }
func (*resettableResource) Resources() {}

func TestStorage_ForEach_VisitsRegisteredResources(t *testing.T) {
	s := NewStorage()
	res := &resettableResource{}
	s.Insert(res)
	s.Insert(&testResourceA{N: 1})

	visited := 0
	s.ForEach(func(v Resources) {
		visited++
		if rr, ok := v.(interface{ Reset() }); ok {
			rr.Reset()
		}
	})

	if visited != 2 {
		t.Errorf("ForEach visited %d resources, want 2", visited)
	}
	if res.resetCalls != 1 {
		t.Errorf("resetCalls = %d, want 1", res.resetCalls)
	}
}

func TestStorage_InsertDynamic_KeyedByConcreteType(t *testing.T) {
	s := NewStorage()
	s.InsertDynamic(&testResourceA{N: 5})

	got, ok := s.TryGet[*testResourceA]()
	if !ok || got.N != 5 {
		t.Errorf("TryGet[*testResourceA]() = %+v, %v, want {5}, true", got, ok)
	}
}

type persistedResource struct{ N int }

func (*persistedResource) Resources()         {}
func (p *persistedResource) Persisted() []any { return []any{&p.N} }

var _ Serializable = (*persistedResource)(nil)

func TestStorage_Persisted_AggregatesSerializableItems(t *testing.T) {
	s := NewStorage()
	s.Insert(&testResourceA{N: 1}) // not Serializable — must be skipped
	s.Insert(&persistedResource{N: 42})

	got := s.Persisted()
	if len(got) != 1 {
		t.Fatalf("Persisted() returned %d pointers, want 1", len(got))
	}
	if *(got[0].(*int)) != 42 {
		t.Errorf("Persisted()[0] = %v, want 42", got[0])
	}
}

type unencodableResource struct{ fn func() }

func (*unencodableResource) Resources()         {}
func (r *unencodableResource) Persisted() []any { return []any{&r.fn} }

var _ Serializable = (*unencodableResource)(nil)

func TestStorage_Insert_PanicsOnUnencodablePersisted(t *testing.T) {
	s := NewStorage()
	defer func() {
		if recover() == nil {
			t.Fatal("expected Insert to panic when Persisted() returns an unencodable value")
		}
	}()
	s.Insert(&unencodableResource{})
}

func TestStorage_InsertDynamic_PanicsOnUnencodablePersisted(t *testing.T) {
	s := NewStorage()
	defer func() {
		if recover() == nil {
			t.Fatal("expected InsertDynamic to panic when Persisted() returns an unencodable value")
		}
	}()
	s.InsertDynamic(&unencodableResource{})
}

func TestStorage_Persisted_DeterministicOrder(t *testing.T) {
	s := NewStorage()
	s.Insert(&persistedResource{N: 1})
	s.Insert(&testResourceB{S: "x"})

	first := s.Persisted()
	second := s.Persisted()
	if len(first) != len(second) {
		t.Fatalf("Persisted() length changed between calls: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("Persisted()[%d] changed between calls: %v vs %v — order must be deterministic", i, first[i], second[i])
		}
	}
}
