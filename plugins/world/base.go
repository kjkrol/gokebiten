package world

// Base is what every entity in the world is made of: where it is, how it moves,
// and which kind it was spawned from.
type Base struct {
	Pos    Position
	Vel    Velocity
	TypeID TypeID
}
