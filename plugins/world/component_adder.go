package world

import (
	"unsafe"

	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

// entityExtras attaches one further component to every entity a populate call spawns.
type entityExtras interface {
	Components() []goke.Addable
	Init(cursor *goke.Cursor, i int, data any, id uid.UID64)
}

// componentAdder is a ready-made entityExtras: sets each entity's T from its Roster Entry's Data.
type componentAdder[T any] struct {
	comp   goke.Comp[T]
	value  func(data any) T
	effect func(v T, id uid.UID64)
}

var _ entityExtras = (*componentAdder[struct{}])(nil)

func (e *componentAdder[T]) Components() []goke.Addable { return []goke.Addable{&e.comp} }

func (e *componentAdder[T]) Init(cursor *goke.Cursor, i int, data any, id uid.UID64) {
	v := e.value(data)
	if unsafe.Sizeof(*new(T)) > 0 {
		e.comp.Slice(cursor)[i] = v
	}
	if e.effect != nil {
		e.effect(v, id)
	}
}
