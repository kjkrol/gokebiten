package collisions_test

import (
	"testing"
	"time"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/plugins/collisions"
)

func TestHit_Zero_HasNoExpiry(t *testing.T) {
	var h collisions.Hit
	if h.HasExpiry() {
		t.Error("expected a zero-value Hit to have no expiry")
	}
	if !h.ExpiresAt().IsZero() {
		t.Errorf("expected ExpiresAt() to be zero, got %v", h.ExpiresAt())
	}
}

func TestHit_SetExpiresAt_RoundTrip(t *testing.T) {
	want := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)

	var h collisions.Hit
	h.SetExpiresAt(want)

	if !h.HasExpiry() {
		t.Error("expected HasExpiry() to be true after SetExpiresAt")
	}
	if got := h.ExpiresAt(); !got.Equal(want) {
		t.Errorf("ExpiresAt() = %v, want %v", got, want)
	}
}

// TestHit_ExpiresAt_RoundTrip_Persistence guards the reason ExpiresAt
// changed from a time.Time field to an int64 (ExpiresAtNano) in the first
// place: goke flags time.Time (BinaryMarshaler-backed) fields as off-chunk,
// and the point of the change was to keep Hit fully in-chunk while a real
// Save/Load round-trip still recovers the exact expiry instant.
func TestHit_ExpiresAt_RoundTrip_Persistence(t *testing.T) {
	path := t.TempDir() + "/save.bin"
	want := time.Date(2026, time.August, 21, 12, 0, 0, 0, time.UTC)

	ecs := goke.New()
	var hit goke.Comp[collisions.Hit]
	ecs.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		f := si.NewFactory(&hit)
		f.Create(1)
		f.Next()
		h := hit.Slice(&f.Cursor)
		h[0].SetExpiresAt(want)
	}})

	ecs.Pause()
	if err := ecs.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ecs2 := goke.New()
	if err := ecs2.Load(path, goke.LoadComp[collisions.Hit]()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	var hit2 goke.Comp[collisions.Hit]
	var q *goke.Query
	ecs2.Setup(goke.SystemFn{OnInit: func(si *goke.SysInit) {
		q = si.NewQueryBuilder(&hit2).Build()
	}})
	q.All()
	found := false
	for q.Next() {
		h := hit2.Slice(q.Cursor())
		for i := range h {
			found = true
			if got := h[i].ExpiresAt(); !got.Equal(want) {
				t.Errorf("ExpiresAt() after Load = %v, want %v", got, want)
			}
		}
	}
	if !found {
		t.Fatal("no entity found after Load")
	}
}
