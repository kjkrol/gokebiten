package navigation

import (
	"math"

	"github.com/kjkrol/aabbworld"
	"github.com/kjkrol/aabbworld/geom"
)

// Direction is a heading in steps of DirectionStep degrees, counter-clockwise from east.
type Direction uint8

// DirectionStep is the angle between neighbouring Directions; Directions counts them round.
const (
	DirectionStep = 15
	Directions    = 360 / DirectionStep
)

const (
	DirE  Direction = 0
	DirNE Direction = 45 / DirectionStep
	DirN  Direction = 90 / DirectionStep
	DirNW Direction = 135 / DirectionStep
	DirW  Direction = 180 / DirectionStep
	DirSW Direction = 225 / DirectionStep
	DirS  Direction = 270 / DirectionStep
	DirSE Direction = 315 / DirectionStep
)

const directionEpsilon = 1e-6

// directionBetween is the nearest Direction from have to want, the short way round a wrapping axis.
func directionBetween(have, want geom.Vec, width, height uint32, edges aabbworld.Edges) Direction {
	dx := shortestAxisDelta(have.X, want.X, width, edges.WrapsX())
	dy := shortestAxisDelta(have.Y, want.Y, height, edges.WrapsY())
	if math.Abs(dx) < directionEpsilon && math.Abs(dy) < directionEpsilon {
		return DirS
	}
	deg := math.Atan2(-dy, dx) * 180 / math.Pi
	k := int(math.Round(deg/DirectionStep)) % Directions
	if k < 0 {
		k += Directions
	}
	return Direction(k)
}

// Angle is the Direction in degrees, counter-clockwise from east.
func (d Direction) Angle() float64 { return float64(d) * DirectionStep }

func opposite(d Direction) Direction { return (d + Directions/2) % Directions }
