package render

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

// AppearanceExtras is a ready-made world.EntityExtras (checked structurally): sets each entity's Appearance via look.
type AppearanceExtras struct {
	fAppear goke.Comp[Appearance]
	look    func(index int) Appearance
}

func NewAppearanceExtras(look func(index int) Appearance) *AppearanceExtras {
	return &AppearanceExtras{look: look}
}

func (e *AppearanceExtras) Components() []goke.Addable {
	return []goke.Addable{&e.fAppear}
}

func (e *AppearanceExtras) Init(cursor *goke.Cursor, i, index int, _ uid.UID64) {
	e.fAppear.Slice(cursor)[i] = e.look(index)
}
