package game

import (
	"reflect"
	"testing"
)

func mustStack(t *testing.T, scenes ...Scene) Stack {
	t.Helper()
	s, err := NewStack(scenes...)
	if err != nil {
		t.Fatalf("NewStack: unexpected error: %v", err)
	}
	return s
}

func TestComposition_ShowIsIdempotentOnPosition(t *testing.T) {
	a := &stubScene{name: "a", focusable: true}
	b := &stubScene{name: "b", focusable: true}
	c := mustStack(t, a, b).Composition()

	c.Show("a")
	c.Show("b")
	c.Show("a") // already visible — must not move to top again

	if got, want := c.Order(), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Order() = %v, want %v", got, want)
	}
}

func TestComposition_ShowIgnoresUnknownName(t *testing.T) {
	c := mustStack(t).Composition()

	c.Show("nope")

	if got := c.Order(); len(got) != 0 {
		t.Errorf("Order() = %v, want empty", got)
	}
}

func TestComposition_Hide(t *testing.T) {
	a := &stubScene{name: "a", focusable: true}
	b := &stubScene{name: "b", focusable: true}
	c := mustStack(t, a, b).Composition()

	c.Show("a")
	c.Show("b")
	c.Hide("a")

	if got, want := c.Order(), []string{"b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Order() = %v, want %v", got, want)
	}

	c.Hide("a") // already hidden — no-op, must not panic
}

func TestComposition_BringToFront(t *testing.T) {
	a := &stubScene{name: "a", focusable: true}
	b := &stubScene{name: "b", focusable: true}
	c := mustStack(t, a, b).Composition()

	c.Show("a")
	c.Show("b")
	c.BringToFront("a")

	if got, want := c.Order(), []string{"b", "a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Order() = %v, want %v", got, want)
	}
}

func TestComposition_BringToFrontNoOpIfNotVisible(t *testing.T) {
	a := &stubScene{name: "a", focusable: true}
	c := mustStack(t, a).Composition()

	c.BringToFront("a") // never shown — must not appear

	if got := c.Order(); len(got) != 0 {
		t.Errorf("Order() = %v, want empty", got)
	}
}

func TestComposition_ActiveSkipsNonFocusableOnTop(t *testing.T) {
	world := &stubScene{name: "world", focusable: true}
	hud := &stubScene{name: "hud", focusable: false}
	c := mustStack(t, world, hud).Composition()

	c.Show("world")
	c.Show("hud") // drawn on top, but must never capture input

	if got, want := c.Active(), "world"; got != want {
		t.Errorf("Active() = %q, want %q", got, want)
	}
}

func TestComposition_ActiveEmptyWhenNothingFocusable(t *testing.T) {
	hud := &stubScene{name: "hud", focusable: false}
	c := mustStack(t, hud).Composition()

	c.Show("hud")

	if got := c.Active(); got != "" {
		t.Errorf("Active() = %q, want empty", got)
	}
}

func TestComposition_Persisted_RoundTripsOrder(t *testing.T) {
	a := &stubScene{name: "a", focusable: true}
	b := &stubScene{name: "b", focusable: true}
	c := mustStack(t, a, b).Composition().(*composition)

	c.Show("b")
	c.Show("a")

	persisted := c.Persisted()
	if len(persisted) != 1 {
		t.Fatalf("Persisted() returned %d targets, want 1", len(persisted))
	}
	orderPtr, ok := persisted[0].(*[]string)
	if !ok {
		t.Fatalf("Persisted()[0] = %T, want *[]string", persisted[0])
	}
	*orderPtr = []string{"a"} // simulate a Load decoding into the same pointer

	if got, want := c.Order(), []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Order() after simulated load = %v, want %v", got, want)
	}
}
