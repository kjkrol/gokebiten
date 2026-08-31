package collisions

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

// CollidableExtras adds Collision to spawned entities — pass it as one of world.Module.Populate's populators.
type CollidableExtras struct{ collision goke.Comp[Collision] }

func NewCollidableExtras() *CollidableExtras { return &CollidableExtras{} }

func (e *CollidableExtras) Components() []goke.Addable             { return []goke.Addable{&e.collision} }
func (e *CollidableExtras) Init(*goke.Cursor, int, int, uid.UID64) {}

// StaticExtras tags spawned entities Static — pass it as one of world.Module.Populate's populators, paired with a Placement.
type StaticExtras struct{ static goke.Comp[Static] }

func NewStaticExtras() *StaticExtras { return &StaticExtras{} }

func (e *StaticExtras) Components() []goke.Addable             { return []goke.Addable{&e.static} }
func (e *StaticExtras) Init(*goke.Cursor, int, int, uid.UID64) {}
