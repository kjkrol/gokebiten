package world

import (
	"unsafe"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

// componentAdder is a ready-made entityExtras: sets each entity's T via value.
type componentAdder[T any] struct {
	comp   goke.Comp[T]
	value  func(index int) T
	effect func(v T, id uid.UID64)
}

var _ entityExtras = (*componentAdder[struct{}])(nil)

func newComponentAdder[T any](value func(index int) T) *componentAdder[T] {
	return &componentAdder[T]{value: value}
}

// WithEffect attaches a side effect run once, right after Init writes T for
// each spawned entity — for state (an occupancy tracker, say) that must
// react to the value as it's set, not just store it.
func (e *componentAdder[T]) WithEffect(effect func(v T, id uid.UID64)) *componentAdder[T] {
	e.effect = effect
	return e
}

func (e *componentAdder[T]) Components() []goke.Addable { return []goke.Addable{&e.comp} }

func (e *componentAdder[T]) Init(cursor *goke.Cursor, i, index int, id uid.UID64) {
	v := e.value(index)
	if unsafe.Sizeof(*new(T)) > 0 {
		e.comp.Slice(cursor)[i] = v
	}
	if e.effect != nil {
		e.effect(v, id)
	}
}
