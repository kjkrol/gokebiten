package collisions

import "time"

// HitExpires overrides the collision engine's default Hit lifetime for
// this entity - attach it to entities that need a different expiry than
// NewPlugin's/New's default.
type HitExpires struct {
	Duration time.Duration
}
