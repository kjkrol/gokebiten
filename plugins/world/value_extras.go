package world

import (
	"unsafe"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

// ValueExtras is a ready-made EntityExtras: sets each entity's T via value.
type ValueExtras[T any] struct {
	comp   goke.Comp[T]
	value  func(index int) T
	effect func(v T, id uid.UID64)
}

var _ EntityExtras = (*ValueExtras[struct{}])(nil)

func NewValueExtras[T any](value func(index int) T) *ValueExtras[T] {
	return &ValueExtras[T]{value: value}
}

// WithEffect attaches a side effect run once, right after Init writes T for
// each spawned entity — for state (an occupancy tracker, say) that must
// react to the value as it's set, not just store it.
func (e *ValueExtras[T]) WithEffect(effect func(v T, id uid.UID64)) *ValueExtras[T] {
	e.effect = effect
	return e
}

func (e *ValueExtras[T]) Components() []goke.Addable { return []goke.Addable{&e.comp} }

func (e *ValueExtras[T]) Init(cursor *goke.Cursor, i, index int, id uid.UID64) {
	v := e.value(index)
	if unsafe.Sizeof(*new(T)) > 0 {
		e.comp.Slice(cursor)[i] = v
	}
	if e.effect != nil {
		e.effect(v, id)
	}
}
