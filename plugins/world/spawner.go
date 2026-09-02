package world

// Spawner decides the starting Position and Velocity of the index-th of
// count entities — pass it directly as Populate's first argument.
type Spawner interface {
	Spawn(index, count int) (Position, Velocity)
}

// SpawnerFunc adapts a plain function to Spawner.
type SpawnerFunc func(index, count int) (Position, Velocity)

func (f SpawnerFunc) Spawn(index, count int) (Position, Velocity) { return f(index, count) }
