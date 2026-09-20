// Package hit is a ready-made reaction to contacts: an entity that struck
// something goes on showing it, long after the single tick its contact lasted.
//
// Hand Show to plugin.Each[hit.Mark], register that with
// collisions.Plugin.RegisterBehavior, and give Mark to whichever kinds should
// show hits — Overlay is the matching draw strategy for world.Renderer. It rides
// the broad phase's own pass, which visits every collidable entity anyway.
package hit

import (
	"time"

	"github.com/kjkrol/gokebiten/plugin"
	"github.com/kjkrol/gokebiten/plugins/collisions"
	"github.com/kjkrol/gokebiten/plugins/world"
)

// Mark is how long this entity shows a hit, and whether one is showing now —
// give it to a kind with k.Const(hit.Mark{Duration: 100 * time.Millisecond}).
type Mark struct {
	Duration      time.Duration
	ExpiresAtNano int64
}

// Active reports whether a hit is showing, as of the last tick the behavior ran.
func (m Mark) Active() bool { return m.ExpiresAtNano != 0 }

// Show marks an entity that struck something, for as long as its own Duration —
// or d, for one that sets none. A Mark whose time has passed is cleared in the
// same pass, once a tick, which is what lets a renderer read Active instead of
// asking the clock for every entity it draws.
func Show(d time.Duration) func(plugin.Tick, *Mark, collisions.Struck) {
	return func(t plugin.Tick, m *Mark, struck collisions.Struck) {
		now := t.Now.UnixNano()
		switch {
		case len(struck.Contacts) > 0:
			m.ExpiresAtNano = now + int64(m.lasting(d))
		case m.Active() && now > m.ExpiresAtNano:
			m.ExpiresAtNano = 0
		}
	}
}

// lasting is the entity's own hit duration, or fallback for one that sets none.
func (m Mark) lasting(fallback time.Duration) time.Duration {
	if m.Duration > 0 {
		return m.Duration
	}
	return fallback
}

// Overlay draws with on top of whatever an entity looks like, while its Mark is active.
func Overlay(with world.Appearance) world.AppearanceStrategy[Mark] {
	return world.AppearanceStrategyFn[Mark](func(dst []world.Appearance, m Mark) []world.Appearance {
		if !m.Active() {
			return dst
		}
		return append(dst, with)
	})
}
