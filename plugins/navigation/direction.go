package navigation

import "github.com/kjkrol/gokg/geom"

type Direction uint8

const (
	DirN Direction = iota
	DirS
	DirE
	DirW
	DirNE
	DirNW
	DirSE
	DirSW
)

const directionEpsilon = 1e-6

func directionBetween(have, want geom.Vec[float64], width, height uint32, toroidal bool) Direction {
	dx := shortestAxisDelta(have.X, want.X, width, toroidal)
	dy := shortestAxisDelta(have.Y, want.Y, height, toroidal)

	switch {
	case dx > directionEpsilon && dy < -directionEpsilon:
		return DirNE
	case dx > directionEpsilon && dy > directionEpsilon:
		return DirSE
	case dx < -directionEpsilon && dy < -directionEpsilon:
		return DirNW
	case dx < -directionEpsilon && dy > directionEpsilon:
		return DirSW
	case dx > directionEpsilon:
		return DirE
	case dx < -directionEpsilon:
		return DirW
	case dy < -directionEpsilon:
		return DirN
	default:
		return DirS
	}
}

func opposite(d Direction) Direction {
	switch d {
	case DirN:
		return DirS
	case DirS:
		return DirN
	case DirE:
		return DirW
	case DirW:
		return DirE
	case DirNE:
		return DirSW
	case DirNW:
		return DirSE
	case DirSE:
		return DirNW
	default:
		return DirNE
	}
}
