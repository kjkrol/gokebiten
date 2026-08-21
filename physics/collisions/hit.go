package collisions

import "time"

// Hit is a temporary tag applied to entities involved in a collision.
// BroadPhase batch-adds it (empty value) to every candidate; NarrowPhase
// either confirms it (sets ExpiresAt) once a real contact resolves, or
// withdraws it (removes the component) if the candidate turned out to be a
// broad-phase false positive. TagExpirySystem removes it later once
// ExpiresAt passes, independent of this confirm/reject step.
type Hit struct {
	// ExpiresAtNano is Hit's storage representation — a plain int64 keeps
	// this component in-chunk. Use ExpiresAt/SetExpiresAt, not this field
	// directly.
	ExpiresAtNano int64
}

// SetExpiresAt sets the expiry to t, which must be non-zero.
func (h *Hit) SetExpiresAt(t time.Time) { h.ExpiresAtNano = t.UnixNano() }

// ExpiresAt returns the expiry set via SetExpiresAt, or the zero time.Time
// if never set.
func (h Hit) ExpiresAt() time.Time {
	if h.ExpiresAtNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, h.ExpiresAtNano)
}

// HasExpiry reports whether SetExpiresAt was ever called, without the
// int64->time.Time conversion ExpiresAt does.
func (h Hit) HasExpiry() bool { return h.ExpiresAtNano != 0 }
