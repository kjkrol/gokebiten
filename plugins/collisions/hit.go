package collisions

import "time"

// Hit is a temporary tag applied to entities involved in a collision —
// NarrowPhase confirms or withdraws it; TagExpirySystem removes it later.
type Hit struct {
	// ExpiresAtNano is Hit's storage representation — a plain int64 keeps
	// this component in-chunk. Use ExpiresAt/SetExpiresAt, not this field
	// directly.
	ExpiresAtNano int64
}

// SetExpiresAt sets the expiry to t, which must be non-zero.
func (h *Hit) SetExpiresAt(t time.Time) { h.ExpiresAtNano = t.UnixNano() }

// ExpiresAt returns the expiry set via SetExpiresAt, or the zero time.Time if never set.
func (h Hit) ExpiresAt() time.Time {
	if h.ExpiresAtNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, h.ExpiresAtNano)
}

// HasExpiry reports whether SetExpiresAt was ever called.
func (h Hit) HasExpiry() bool { return h.ExpiresAtNano != 0 }
