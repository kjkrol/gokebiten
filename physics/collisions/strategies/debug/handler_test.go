package debug_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kjkrol/gokebiten/physics/collisions"
	"github.com/kjkrol/gokebiten/physics/collisions/strategies/debug"
	"github.com/kjkrol/uid"
)

func TestHandler_DefaultFormat_WritesEntityIDs(t *testing.T) {
	var buf bytes.Buffer
	h := debug.NewHandler(debug.WithWriter(&buf))

	h.OnCollision(nil, collisions.CollisionEvent{EntityA: uid.UID64(1), EntityB: uid.UID64(2)})

	got := buf.String()
	if !strings.Contains(got, "1") || !strings.Contains(got, "2") {
		t.Errorf("output = %q, want it to mention both entity ids", got)
	}
}

func TestHandler_WithFormat_UsesCustomFormatter(t *testing.T) {
	var buf bytes.Buffer
	h := debug.NewHandler(
		debug.WithWriter(&buf),
		debug.WithFormat(func(e collisions.CollisionEvent) string { return "custom-line" }),
	)

	h.OnCollision(nil, collisions.CollisionEvent{})

	if got := buf.String(); strings.TrimSpace(got) != "custom-line" {
		t.Errorf("output = %q, want %q", got, "custom-line")
	}
}
