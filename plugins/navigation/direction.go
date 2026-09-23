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

const directionEpsilon = 1e-6

// DirectionAt is the Direction nearest deg degrees counter-clockwise from east.
func DirectionAt(deg float64) Direction {
	k := int(math.Round(deg/DirectionStep)) % Directions
	if k < 0 {
		k += Directions
	}
	return Direction(k)
}

// directionBetween is the nearest Direction from have to want, the short way round a wrapping axis.
func directionBetween(have, want geom.Vec, width, height uint32, edges aabbworld.Edges) Direction {
	dx := shortestAxisDelta(have.X, want.X, width, edges.WrapsX())
	dy := shortestAxisDelta(have.Y, want.Y, height, edges.WrapsY())
	if math.Abs(dx) < directionEpsilon && math.Abs(dy) < directionEpsilon {
		return DirectionAt(270) // straight down: a placeholder for a step of no length
	}
	return DirectionAt(math.Atan2(-dy, dx) * 180 / math.Pi)
}

// Angle is the Direction in degrees, counter-clockwise from east.
func (d Direction) Angle() float64 { return float64(d) * DirectionStep }

func opposite(d Direction) Direction { return (d + Directions/2) % Directions }
