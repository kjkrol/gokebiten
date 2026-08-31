package world

import (
	"github.com/kjkrol/goke/v3"
	"github.com/kjkrol/uid"
)

// Placement decides where the index-th of count entities starts — pass it
// directly as Populate's first populator for velocity-less entities.
type Placement interface {
	Place(index, count int) Position
	Components() []goke.Addable
	Init(cursor *goke.Cursor, i, index int, id uid.UID64)
}

// Motion decides the starting velocity of the index-th of count entities.
type Motion interface {
	InitialVelocity(index, count int) Velocity
}

// Spawner decides the starting Position and Velocity of the index-th of
// count entities — pass it directly as Populate's first populator.
type Spawner interface {
	Spawn(index, count int) (Position, Velocity)
	Components() []goke.Addable
	Init(cursor *goke.Cursor, i, index int, id uid.UID64)
}

// NewSpawner composes an independently-named Placement and Motion into a Spawner.
func NewSpawner(placement Placement, motion Motion) Spawner {
	return &spawner{placement: placement, motion: motion}
}

type spawner struct {
	placement Placement
	motion    Motion
}

func (s *spawner) Spawn(index, count int) (Position, Velocity) {
	return s.placement.Place(index, count), s.motion.InitialVelocity(index, count)
}
func (s *spawner) Components() []goke.Addable             { return nil }
func (s *spawner) Init(*goke.Cursor, int, int, uid.UID64) {}
