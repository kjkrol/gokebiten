package gokebiten

import "testing"

type testResourceA struct{ N int }
type testResourceB struct{ S string }

func TestResources_InsertGet_RoundTrip(t *testing.T) {
	r := NewResources()
	r.insertResource(&testResourceA{N: 7})

	got := r.Get[*testResourceA]()
	if got.N != 7 {
		t.Errorf("GetResource[*testResourceA]().N = %d, want 7", got.N)
	}
}

func TestResources_TryGetResource_MissingReturnsFalse(t *testing.T) {
	r := NewResources()
	_, ok := r.TryGet[*testResourceA]()
	if ok {
		t.Error("TryGetResource on an empty registry: ok = true, want false")
	}
}

func TestResources_GetResource_MissingPanics(t *testing.T) {
	r := NewResources()
	defer func() {
		if recover() == nil {
			t.Fatal("expected GetResource to panic when the resource isn't registered")
		}
	}()
	r.Get[*testResourceA]()
}

func TestResources_InsertResource_OverwritesPreviousValue(t *testing.T) {
	r := NewResources()
	r.insertResource(&testResourceA{N: 1})
	r.insertResource(&testResourceA{N: 2})

	got := r.Get[*testResourceA]()
	if got.N != 2 {
		t.Errorf("GetResource[*testResourceA]().N = %d, want 2 (last InsertResource should win)", got.N)
	}
}

func TestResources_DifferentTypesDoNotCollide(t *testing.T) {
	r := NewResources()
	r.insertResource(&testResourceA{N: 1})
	r.insertResource(&testResourceB{S: "hi"})

	if got := r.Get[*testResourceA](); got.N != 1 {
		t.Errorf("testResourceA.N = %d, want 1", got.N)
	}
	if got := r.Get[*testResourceB](); got.S != "hi" {
		t.Errorf("testResourceB.S = %q, want %q", got.S, "hi")
	}
}

type resettableResource struct{ resetCalls int }

func (r *resettableResource) Reset() { r.resetCalls++ }

func TestResources_ForEach_VisitsRegisteredResources(t *testing.T) {
	r := NewResources()
	res := &resettableResource{}
	r.insertResource(res)
	r.insertResource(&testResourceA{N: 1})

	visited := 0
	r.forEach(func(v any) {
		visited++
		if rr, ok := v.(Resettable); ok {
			rr.Reset()
		}
	})

	if visited != 2 {
		t.Errorf("forEach visited %d resources, want 2", visited)
	}
	if res.resetCalls != 1 {
		t.Errorf("resetCalls = %d, want 1", res.resetCalls)
	}
}
