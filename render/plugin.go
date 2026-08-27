package render

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/gokebiten/physics/kinematics"
)

// Plugin computes or refines an entity's draw layers — runs in the order
// registered, each seeing what earlier plugins left in dst.
type Plugin interface {
	Bind(qb *goke.QueryBuilder)
	Apply(cur *goke.Cursor, i int, dst []Appearance) []Appearance
}

// Strategy decides how to fold override T into an entity's draw layers.
type Strategy[T any] interface {
	Resolve(dst []Appearance, override T) []Appearance
}

// StrategyFn adapts a plain function to Strategy[T].
type StrategyFn[T any] func(dst []Appearance, override T) []Appearance

func (f StrategyFn[T]) Resolve(dst []Appearance, override T) []Appearance {
	return f(dst, override)
}

// Replace returns a Strategy setting dst[0] to with.
func Replace[T any](with Appearance) Strategy[T] {
	return StrategyFn[T](func(dst []Appearance, _ T) []Appearance {
		dst[0] = with
		return dst[:1]
	})
}

// Overlay returns a Strategy appending with on top of dst.
func Overlay[T any](with Appearance) Strategy[T] {
	return StrategyFn[T](func(dst []Appearance, _ T) []Appearance {
		return append(dst, with)
	})
}

// Modify returns a Strategy replacing dst[0] with f(dst[0], override).
func Modify[T any](f func(Appearance, T) Appearance) Strategy[T] {
	return StrategyFn[T](func(dst []Appearance, override T) []Appearance {
		dst[0] = f(dst[0], override)
		return dst
	})
}

// Component binds one optionally-tracked component type T to the Strategy
// resolving it — pass &Component[T]{...} to NewEntitiesRenderer, one per tracked "kind".
type Component[T any] struct {
	comp     goke.OptComp[T]
	Strategy Strategy[T]
}

func (h *Component[T]) Bind(qb *goke.QueryBuilder) { qb.Optional(&h.comp) }

func (h *Component[T]) Apply(cur *goke.Cursor, i int, dst []Appearance) []Appearance {
	if !h.comp.Present(cur) {
		return dst
	}
	return h.Strategy.Resolve(dst, h.comp.Slice(cur)[i])
}

// FacingFromVelocity returns a Plugin orienting the sprite by the entity's
// current Velocity — every entity has one, so this always applies.
func FacingFromVelocity(spriteFor func(kinematics.Velocity) uint8) Plugin {
	return &Component[kinematics.Velocity]{
		Strategy: Modify(func(a Appearance, v kinematics.Velocity) Appearance {
			a.SpriteID = spriteFor(v)
			return a
		}),
	}
}
