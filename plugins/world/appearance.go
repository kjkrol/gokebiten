package world

import (
	"image/color"

	"github.com/kjkrol/goke/v3"
)

type Appearance struct {
	Color    color.RGBA
	SpriteID uint8
}

// AppearanceStrategy decides how to fold override T into an entity's draw layers.
type AppearanceStrategy[T any] interface {
	Resolve(dst []Appearance, override T) []Appearance
}

// AppearanceStrategyFn adapts a plain function to AppearanceStrategy[T].
type AppearanceStrategyFn[T any] func(dst []Appearance, override T) []Appearance

func (f AppearanceStrategyFn[T]) Resolve(dst []Appearance, override T) []Appearance {
	return f(dst, override)
}

// replace returns a Strategy setting dst[0] to with.
func replace[T any](with Appearance) AppearanceStrategy[T] {
	return AppearanceStrategyFn[T](func(dst []Appearance, _ T) []Appearance {
		dst[0] = with
		return dst[:1]
	})
}

// overlay returns a Strategy appending with on top of dst.
func overlay[T any](with Appearance) AppearanceStrategy[T] {
	return AppearanceStrategyFn[T](func(dst []Appearance, _ T) []Appearance {
		return append(dst, with)
	})
}

// modify returns a Strategy replacing dst[0] with f(dst[0], override).
func modify[T any](f func(Appearance, T) Appearance) AppearanceStrategy[T] {
	return AppearanceStrategyFn[T](func(dst []Appearance, override T) []Appearance {
		dst[0] = f(dst[0], override)
		return dst
	})
}

// Facing returns a Strategy rewriting SpriteID from the entity's current Velocity, via spriteFor.
func Facing(spriteFor func(Velocity) uint8) AppearanceStrategy[Velocity] {
	return modify(func(a Appearance, v Velocity) Appearance {
		a.SpriteID = spriteFor(v)
		return a
	})
}

// AppearanceModifier computes or refines an entity's draw layers — runs in
// the order registered, each seeing what earlier ones left in dst.
type AppearanceModifier = Modifier[[]Appearance]

var _ AppearanceModifier = (*conditionalApperanceStrategy[struct{}])(nil)

type conditionalApperanceStrategy[T any] struct {
	comp     goke.OptComp[T]
	Strategy AppearanceStrategy[T]
}

func (h *conditionalApperanceStrategy[T]) Bind(qb *goke.QueryBuilder) { qb.Optional(&h.comp) }

func (h *conditionalApperanceStrategy[T]) Apply(cur *goke.Cursor, i int, dst []Appearance) []Appearance {
	if !h.comp.Present(cur) {
		return dst
	}
	return h.Strategy.Resolve(dst, h.comp.Slice(cur)[i])
}
